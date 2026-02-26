# 🖥️ nvtop Prometheus Exporter

A lightweight Prometheus exporter that scrapes GPU metrics from `nvtop -s` and exposes them in Prometheus format.

## 📋 Overview

This exporter calls `nvtop -s` on each Prometheus scrape, parses the JSON output, and serves the metrics on a local HTTP endpoint. It handles multi-GPU setups automatically via the `device` label.

> ⚠️ **Note:** `nvtop 3.3.1` produces malformed JSON (missing commas between fields). This exporter includes a sanitizer that fixes the output before parsing. The bug was fixed in `nvtop 3.3.2`.

## 📊 Exposed Metrics

| Metric | Description | Unit |
|---|---|---|
| `nvtop_gpu_clock_mhz` | GPU core clock speed | MHz |
| `nvtop_mem_clock_mhz` | Memory clock speed | MHz |
| `nvtop_temperature_celsius` | GPU temperature | °C |
| `nvtop_fan_speed_percent` | Fan speed | % |
| `nvtop_power_draw_watts` | Power consumption | W |
| `nvtop_gpu_utilization_percent` | GPU utilization | % |
| `nvtop_mem_utilization_percent` | Memory utilization | % |
| `nvtop_mem_total_bytes` | Total GPU memory | Bytes |
| `nvtop_mem_used_bytes` | Used GPU memory | Bytes |
| `nvtop_mem_free_bytes` | Free GPU memory | Bytes |

All metrics are labeled with `device` (e.g. `NVIDIA GeForce RTX 3090`).

An example looks like this:

```
# HELP nvtop_fan_speed_percent Fan speed in percent
# TYPE nvtop_fan_speed_percent gauge
nvtop_fan_speed_percent{device="NVIDIA GeForce RTX 3090"} 0
# HELP nvtop_gpu_clock_mhz GPU clock speed in MHz
# TYPE nvtop_gpu_clock_mhz gauge
nvtop_gpu_clock_mhz{device="NVIDIA GeForce RTX 3090"} 210
# HELP nvtop_gpu_utilization_percent GPU utilization in percent
# TYPE nvtop_gpu_utilization_percent gauge
nvtop_gpu_utilization_percent{device="NVIDIA GeForce RTX 3090"} 0
# HELP nvtop_mem_clock_mhz Memory clock speed in MHz
# TYPE nvtop_mem_clock_mhz gauge
nvtop_mem_clock_mhz{device="NVIDIA GeForce RTX 3090"} 405
# HELP nvtop_mem_free_bytes Free GPU memory in bytes
# TYPE nvtop_mem_free_bytes gauge
nvtop_mem_free_bytes{device="NVIDIA GeForce RTX 3090"} 2.5295060992e+10
# HELP nvtop_mem_total_bytes Total GPU memory in bytes
# TYPE nvtop_mem_total_bytes gauge
nvtop_mem_total_bytes{device="NVIDIA GeForce RTX 3090"} 2.5769803776e+10
# HELP nvtop_mem_used_bytes Used GPU memory in bytes
# TYPE nvtop_mem_used_bytes gauge
nvtop_mem_used_bytes{device="NVIDIA GeForce RTX 3090"} 4.74742784e+08
# HELP nvtop_mem_utilization_percent Memory utilization in percent
# TYPE nvtop_mem_utilization_percent gauge
nvtop_mem_utilization_percent{device="NVIDIA GeForce RTX 3090"} 1
# HELP nvtop_power_draw_watts Power draw in watts
# TYPE nvtop_power_draw_watts gauge
nvtop_power_draw_watts{device="NVIDIA GeForce RTX 3090"} 15
# HELP nvtop_temperature_celsius GPU temperature in Celsius
# TYPE nvtop_temperature_celsius gauge
nvtop_temperature_celsius{device="NVIDIA GeForce RTX 3090"} 38
```

## 🔧 Prerequisites

- `nvtop` installed and available in `$PATH`
- NVIDIA GPU with working drivers
- NVIDIA Container Toolkit (if run inside a container)
- Go 1.21+ (to build the project)

## 🚀 Build & Run

```bash
git clone <repo-url> && cd nvtop-exporter
go mod tidy
go build -o nvtop-exporter .
./nvtop-exporter
```

The exporter listens on `127.0.0.1:9000/nvmetrics`.

```bash
curl -s localhost:9000/nvmetrics
```

### Local Docker Image Build

```bash
git clone https://github.com/Syllo/nvtop.git
docker build -t nvtop:local nvtop/
docker build -t nvtop-exporter --build-arg BASE_IMAGE=nvtop:local .
```

## 🔗 Prometheus Scrape Configuration

Add the following scrape job to your `prometheus.yml`:


```yaml
scrape_configs:
  - job_name: "nvtop"
    metrics_path: "/nvmetrics"
    static_configs:
      - targets: ["<host>:9000"]
```

## 🐳 Run as container

To run this in a container the nvidia container toolkit
needs to be installed on the host system.

```yaml
services:
  nvtop-exporter:
    image: ghcr.io/virtuos/nvtop-exporter:latest
    restart: always
    runtime: nvidia
```

## 📦 Run as systemd Service

Create `/etc/systemd/system/nvtop-exporter.service`:

```ini
[Unit]
Description=nvtop Prometheus Exporter
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nvtop-exporter
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo cp nvtop-exporter /usr/local/bin/
sudo systemctl daemon-reload
sudo systemctl enable --now nvtop-exporter
```

## 📝 License

MIT

## Authors

virtUOS, Osnabrueck University
