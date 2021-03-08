# Collector for OpenWhisk Function Executions

This collector receives traces from OpenWhisk function executions and enriches them with meta-information such as initialization and wait time.

## Installation

1. Clone <https://github.com/koelschkellerkind/opentelemetry-collector-builder>
2. Create the binary of the builder with

```go
go build
```

3. Add the binary to your your PATH

4. Create the binary of the collector with

```bash
opentelemetry-collector-builder --config builder.yaml
```

5.

```bash
cd owtracecollector-dist
```

6. Start the collector with

```bash
./owtracecollector --config ../config.yaml  
```

## Pitfalls

* Using tabs in the builder YAML results in an error
