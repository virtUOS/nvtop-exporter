package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// nvtopDevice maps the JSON output of nvtop -s
type nvtopDevice struct {
	DeviceName string `json:"device_name"`
	GPUClock   string `json:"gpu_clock"`
	MemClock   string `json:"mem_clock"`
	Temp       string `json:"temp"`
	FanSpeed   string `json:"fan_speed"`
	PowerDraw  string `json:"power_draw"`
	GPUUtil    string `json:"gpu_util"`
	MemUtil    string `json:"mem_util"`
	MemTotal   string `json:"mem_total"`
	MemUsed    string `json:"mem_used"`
	MemFree    string `json:"mem_free"`
}

// nvidia-smi XML structures
type nvidiaSmiLog struct {
	XMLName       xml.Name    `xml:"nvidia_smi_log"`
	DriverVersion string      `xml:"driver_version"`
	CudaVersion   string      `xml:"cuda_version"`
}

// stripUnit removes known suffixes (MHz, C, %, W) and converts to float64.
func stripUnit(s string) float64 {
	s = strings.TrimSpace(s)
	for _, suffix := range []string{"MHz", "C", "%", "W"} {
		s = strings.TrimSuffix(s, suffix)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("warn: could not parse %q: %v", s, err)
		return 0
	}
	return v
}

// nvtopCollector implements prometheus.Collector
type nvtopCollector struct {
	gpuClock   *prometheus.Desc
	memClock   *prometheus.Desc
	temp       *prometheus.Desc
	fanSpeed   *prometheus.Desc
	powerDraw  *prometheus.Desc
	gpuUtil    *prometheus.Desc
	memUtil    *prometheus.Desc
	memTotal   *prometheus.Desc
	memUsed    *prometheus.Desc
	memFree    *prometheus.Desc
	driverInfo *prometheus.Desc
}

func newNvtopCollector() *nvtopCollector {
	labels := []string{"device"}
	ns := "nvtop"
	return &nvtopCollector{
		gpuClock:  prometheus.NewDesc(ns+"_gpu_clock_mhz", "GPU clock speed in MHz", labels, nil),
		memClock:  prometheus.NewDesc(ns+"_mem_clock_mhz", "Memory clock speed in MHz", labels, nil),
		temp:      prometheus.NewDesc(ns+"_temperature_celsius", "GPU temperature in Celsius", labels, nil),
		fanSpeed:  prometheus.NewDesc(ns+"_fan_speed_percent", "Fan speed in percent", labels, nil),
		powerDraw: prometheus.NewDesc(ns+"_power_draw_watts", "Power draw in watts", labels, nil),
		gpuUtil:   prometheus.NewDesc(ns+"_gpu_utilization_percent", "GPU utilization in percent", labels, nil),
		memUtil:   prometheus.NewDesc(ns+"_mem_utilization_percent", "Memory utilization in percent", labels, nil),
		memTotal:  prometheus.NewDesc(ns+"_mem_total_bytes", "Total GPU memory in bytes", labels, nil),
		memUsed:   prometheus.NewDesc(ns+"_mem_used_bytes", "Used GPU memory in bytes", labels, nil),
		memFree:   prometheus.NewDesc(ns+"_mem_free_bytes", "Free GPU memory in bytes", labels, nil),
		driverInfo: prometheus.NewDesc(
			ns+"_nvidia_driver_info",
			"NVIDIA driver and CUDA version info",
			[]string{"driver_version", "cuda_version"}, nil,
		),
	}
}

func (c *nvtopCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.gpuClock
	ch <- c.memClock
	ch <- c.temp
	ch <- c.fanSpeed
	ch <- c.powerDraw
	ch <- c.gpuUtil
	ch <- c.memUtil
	ch <- c.memTotal
	ch <- c.memUsed
	ch <- c.memFree
	ch <- c.driverInfo
}

func (c *nvtopCollector) Collect(ch chan<- prometheus.Metric) {
	devices, err := queryNvtop()
	if err != nil {
		log.Printf("error: nvtop query failed: %v", err)
		return
	}

	for _, d := range devices {
		name := d.DeviceName
		ch <- prometheus.MustNewConstMetric(c.gpuClock, prometheus.GaugeValue, stripUnit(d.GPUClock), name)
		ch <- prometheus.MustNewConstMetric(c.memClock, prometheus.GaugeValue, stripUnit(d.MemClock), name)
		ch <- prometheus.MustNewConstMetric(c.temp, prometheus.GaugeValue, stripUnit(d.Temp), name)
		ch <- prometheus.MustNewConstMetric(c.fanSpeed, prometheus.GaugeValue, stripUnit(d.FanSpeed), name)
		ch <- prometheus.MustNewConstMetric(c.powerDraw, prometheus.GaugeValue, stripUnit(d.PowerDraw), name)
		ch <- prometheus.MustNewConstMetric(c.gpuUtil, prometheus.GaugeValue, stripUnit(d.GPUUtil), name)
		ch <- prometheus.MustNewConstMetric(c.memUtil, prometheus.GaugeValue, stripUnit(d.MemUtil), name)
		ch <- prometheus.MustNewConstMetric(c.memTotal, prometheus.GaugeValue, stripUnit(d.MemTotal), name)
		ch <- prometheus.MustNewConstMetric(c.memUsed, prometheus.GaugeValue, stripUnit(d.MemUsed), name)
		ch <- prometheus.MustNewConstMetric(c.memFree, prometheus.GaugeValue, stripUnit(d.MemFree), name)
	}

	// NVIDIA-specific info metrics (best-effort, skip silently if nvidia-smi is unavailable)
	smi, err := queryNvidiaSmi()
	if err != nil {
		log.Printf("info: nvidia-smi not available, skipping nvidia metrics: %v", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.driverInfo, prometheus.GaugeValue, 1,
		smi.DriverVersion, smi.CudaVersion,
	)

}

// This needs to be done due to a bug in nvtop 3.3.1 that is fixed in 3.3.2.
// missingCommaRe matches a closing `"` at end of a field line
// followed by a newline and another `"` starting the next field — without a comma.
var missingCommaRe = regexp.MustCompile(`"[ \t]*\n([ \t]*")`)

func sanitizeJSON(raw []byte) []byte {
	return missingCommaRe.ReplaceAll(raw, []byte("\",\n$1"))
}

func queryNvtop() ([]nvtopDevice, error) {
	out, err := exec.Command("nvtop", "-s").Output()
	if err != nil {
		return nil, err
	}

	fixed := sanitizeJSON(out) // <-- fix broken JSON before parsing

	var devices []nvtopDevice
	if err := json.Unmarshal(fixed, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

func queryNvidiaSmi() (*nvidiaSmiLog, error) {
	out, err := exec.Command("nvidia-smi", "-q", "-x").Output()
	if err != nil {
		return nil, err
	}

	var smi nvidiaSmiLog
	if err := xml.Unmarshal(out, &smi); err != nil {
		return nil, fmt.Errorf("xml parse error: %w", err)
	}
	return &smi, nil
}

func main() {
	collector := newNvtopCollector()
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	http.Handle("/nvmetrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	log.Println("nvtop exporter listening on :9000/nvmetrics")
	log.Fatal(http.ListenAndServe("0.0.0.0:9000", nil))
}
