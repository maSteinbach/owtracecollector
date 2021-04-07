# OpenWhisk Trace Collector

The collector is configurable with two custom processor: owspanprocessor and owspanattacher.

* __owspanprocessor__ receives spans from an instrumented OpenWhisk function. Each span represents a function execution and must contain the Activation ID of this as an attribute. The processor uses the Activation ID to get further meta information about the function execution via an API call to OpenWhisk. This meta information, the waitTime and initTime of execution, are added as attributes to the span.
* __owspanattacher__ must be located in the pipeline after the owspanprocessor. This is necessary because this processor extracts the waitTime and initTime attributes from the span, and creates corresponding child spans. In total the processor generates a child span each for the function execution, the waitTime and the initTime.

## Example Configuration

```yaml
processors:
    owspanprocessor:
        ow_host: http://owdev-nginx.openwhisk.svc.cluster.local:80
        ow_auth_token: 23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP
        logging: true
    owspanattacher:
        logging: true
```

## Build a Docker Image of the Collector

```bash
docker build -t koelschkellerkind/owtracecollector:tag .
```

## Releases

* 1.0.0 initial release
* 1.0.1 refactoring: owtracecollector is separated in owspanprocessor and owspanattacher (24th March 2021)
* 1.0.2 changed unit of wait and initTime attributes to milliseconds (29th March 2021)
* 1.0.3 renamed child spans of attacher (31st March 2021)
* 1.0.4 bugfix: `start` from OpenWhisk activation record represents beginning of initialization (7th April 2021)

---

## Build a Collector Executable via Opentelemetry-Collector-Builder (DEPRECATED)

Opentelemetry-collector-builder can be used to build a executable of the collector.

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
