# Metrics Collector - Standalone Metrics Collection Tool

A standalone tool for collecting metrics from HTTP endpoints (e.g., Prometheus metrics endpoints). This is a focused version of the metrics functionality.

## Installation

Build from source:

```bash
go build -o bin/metrics-collector ./cmd/metrics-collector
```

Or install globally:

```bash
go install ./cmd/metrics-collector
```

## Usage

```bash
metrics-collector -endpoint <url> [options]
```

**Options:**
- `-endpoint` (required): Metrics endpoint URL (e.g., http://localhost:8080/metrics)
- `-format`: Output format - `prometheus` or `json` (default: prometheus)
- `-output`: Output file (default: stdout)
- `-interval`: Collection interval (0 = single collection, e.g., 30s for continuous)

**Examples:**

```bash
# Single collection to stdout
metrics-collector -endpoint http://localhost:8080/metrics

# Save to file
metrics-collector -endpoint http://localhost:8080/metrics -output metrics.txt

# Continuous collection every 30 seconds
metrics-collector -endpoint http://localhost:8080/metrics -interval 30s

# Continuous collection with file output
metrics-collector -endpoint http://localhost:8080/metrics -interval 1m -output metrics.log
```

## Use Cases

### Metrics Backup

```bash
# Backup metrics every hour
metrics-collector -endpoint http://localhost:8080/metrics -output "metrics-$(date +%Y%m%d-%H%M%S).txt"
```

### Continuous Monitoring

```bash
# Collect metrics every minute to a log file
metrics-collector -endpoint http://localhost:8080/metrics -interval 1m -output metrics.log
```

### Metrics Export

```bash
# Export metrics for analysis
metrics-collector -endpoint http://localhost:8080/metrics -output metrics-export.txt
```

### Integration with Monitoring Systems

```bash
#!/bin/bash
# Collect metrics and send to monitoring system
metrics-collector -endpoint http://localhost:8080/metrics -output /tmp/metrics.txt
# Process and send metrics...
```

## Differences from devopscli

- `metrics-collector` is a standalone, focused tool for metrics collection only
- `devopscli metrics` is part of a larger CLI tool with multiple commands
- Both use similar functionality but `metrics-collector` provides continuous collection
- Choose `metrics-collector` for dedicated metrics collection workflows
- Choose `devopscli` for workflows that need multiple DevOps operations

## Future Enhancements

Planned features:
- Metrics filtering and aggregation
- Multiple endpoint support
- Format conversion (Prometheus ↔ JSON)
- Metrics validation
- Integration with monitoring backends (InfluxDB, Prometheus, etc.)

