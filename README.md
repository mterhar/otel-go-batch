# Batch Job Example for OpenTelemetry using Go

This repo is an example for how to make tracing work well for an exotic trace type, batch jobs.

## Run it

To run it locally, export some environment variables

```shell
OTEL_SERVICE_NAME=otel-go-batch
OTEL_EXPORTER_OTLP_ENDPOINT="https://api.honeycomb.io:443"
OTEL_EXPORTER_OTLP_HEADERS="x-honeycomb-team=[your environment api key]"
```

<!-- OSS metadata badge - rename repo link and set status in OSSMETADATA -->
<!-- [![OSS Lifecycle](https://img.shields.io/osslifecycle/honeycombio/otel-go-batch)](https://github.com/honeycombio/home/blob/main/honeycomb-oss-lifecycle-and-practices.md) -->
