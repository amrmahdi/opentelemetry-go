module go.opentelemetry.io/otel/example/grpc

go 1.13

replace go.opentelemetry.io/otel => ../..

replace go.opentelemetry.io/otel/exporter/trace/jaeger => ../../exporter/trace/jaeger

require (
	github.com/golang/protobuf v1.3.2
	go.opentelemetry.io/otel v0.2.1
	go.opentelemetry.io/otel/exporter/trace/jaeger v1.0.0
	google.golang.org/grpc v1.24.0
)
