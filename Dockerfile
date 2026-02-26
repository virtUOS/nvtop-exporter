ARG BASE_IMAGE=nvtop:local

# ---- Build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN go build -o nvtop-exporter .

# ---- Runtime stage ----
FROM $BASE_IMAGE

COPY --from=builder /src/nvtop-exporter /usr/local/bin/

EXPOSE 9000
ENTRYPOINT ["/usr/local/bin/nvtop-exporter"]
