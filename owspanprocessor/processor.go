package owspanprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/apache/openwhisk-client-go/whisk"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

var (
	_       component.TracesProcessor = (*owSpanProcessor)(nil)
	retries int                       = 2
	sleep   time.Duration             = 5
)

type owSpanProcessor struct {
	next     consumer.TracesConsumer
	owclient *whisk.Client
	logger   *zap.Logger
	logging  bool
}

func newOwSpanProcessor(next consumer.TracesConsumer, cfg *Config, logger *zap.Logger) (*owSpanProcessor, error) {
	config := &whisk.Config{
		Host:      cfg.OwHost,
		AuthToken: cfg.OwAuthToken,
		Insecure:  true,
	}
	// Whisk API does not return an error for invalid host or token
	c, err := whisk.NewClient(http.DefaultClient, config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to OpenWhisk API of host %s and token %s, error message: %s", cfg.OwHost, cfg.OwAuthToken, err.Error())
	}
	// Check if the connection is valid
	_, _, err = c.Actions.Get("", false)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to OpenWhisk API of host %s and token %s, error message: %s", cfg.OwHost, cfg.OwAuthToken, err.Error())
	}
	return &owSpanProcessor{next: next, owclient: c, logger: logger, logging: cfg.Logging}, nil
}

func (p *owSpanProcessor) ConsumeTraces(ctx context.Context, batch pdata.Traces) error {
	// ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]
	for i := 0; i < batch.ResourceSpans().Len(); i++ {
		rs := batch.ResourceSpans().At(i)
		for j := 0; j < rs.InstrumentationLibrarySpans().Len(); j++ {
			spans := rs.InstrumentationLibrarySpans().At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				id, ok := span.Attributes().Get("activationId")
				if !ok && p.logging {
					p.logger.Info("Span " + span.SpanID().HexString() + " without activation id attribute")
				} else {
					if p.logging {
						p.logger.Info("Processing span " + span.SpanID().HexString() + " with activation id: " + id.StringVal())
					}
					activation, res, err := p.owclient.Activations.Get(id.StringVal())
					counter := retries
					for err != nil && counter > 0 {
						time.Sleep(sleep * time.Second)
						if p.logging {
							p.logger.Info("Access to OpenWhisk API failed for span " + span.SpanID().HexString() + " with activation id: " + id.StringVal() + ", status code: " + res.Status + ", error message: " + err.Error() + ". Retrying ...")
						}
						activation, res, err = p.owclient.Activations.Get(id.StringVal())
						counter--
					}
					if res.StatusCode == http.StatusOK {
						// OpenTelemetry works with nanoseconds, whereas the OpenWhisk API milliseconds returns.
						executionStartNano := activation.Start * 1e06
						executionEndNano := activation.End * 1e06
						waitTime := activation.Annotations.GetValue("waitTime")
						initTime := activation.Annotations.GetValue("initTime")

						var waitTimeNano int64
						if waitTime != nil {
							waitTimeNano, _ = waitTime.(json.Number).Int64()
							waitTimeNano *= 1e06
							if p.logging {
								p.logger.Info("Span " + span.SpanID().HexString() + " with waitTime: " + strconv.FormatInt(waitTimeNano, 10) + "ns")
							}
						}
						var initTimeNano int64
						if initTime != nil {
							initTimeNano, _ = initTime.(json.Number).Int64()
							initTimeNano *= 1e6
							if p.logging {
								p.logger.Info("Span " + span.SpanID().HexString() + " with initTime: " + strconv.FormatInt(initTimeNano, 10) + "ns")
							}
						}
						// Adjust the start time of the span such that the wait and init time is included. Adjust the end time of the
						// span such that it is equal to the end time received from the OpenWhisk API.
						span.SetStartTime(pdata.TimestampUnixNano(executionStartNano - waitTimeNano - initTimeNano))
						span.SetEndTime(pdata.TimestampUnixNano(executionEndNano))
						// Add waitTime and initTime attributes if present.
						if waitTime != nil {
							waitTimeMilli := waitTimeNano / 1e6
							span.Attributes().InsertInt("waitTimeMilli", waitTimeMilli)
						}
						if initTime != nil {
							initTimeMilli := initTimeNano / 1e6
							span.Attributes().InsertInt("initTimeMilli", initTimeMilli)
						}
					} else {
						if p.logging {
							p.logger.Info("Unable to access OpenWhisk API for span " + span.SpanID().HexString() + " with activation id: " + id.StringVal() + ", status code: " + res.Status + ", error message: " + err.Error())
						}
					}
				}
			}
		}
	}
	return p.next.ConsumeTraces(ctx, batch)
}

func (p *owSpanProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (p *owSpanProcessor) Start(_ context.Context, host component.Host) error {
	return nil
}

func (p *owSpanProcessor) Shutdown(context.Context) error {
	return nil
}