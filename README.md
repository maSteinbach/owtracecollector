# Collector for OpenWhisk Function Executions

This collector receives traces from OpenWhisk function executions and enriches them with meta-information such as initialization and wait time.

## Build a Docker Image of the Collector

```bash
docker build -t koelschkellerkind/owtracecollector .
```

## Build a Collector Executable via Opentelemetry-Collector-Builder

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

5. Then

    ```bash
    cd owtracecollector-dist
    ```

6. Start the collector with

    ```bash
    ./owtracecollector --config ../otel-collector-config.yaml  
    ```

### Pitfalls

* Using tabs in the `builder.yaml` results in an error

### TODO

* Remove OpenWhisk config from `processor.go` line 32-36
* Move `client, err := whisk.NewClient(http.DefaultClient, config)` to Start routine
