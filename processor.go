package owtraceprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/apache/openwhisk-client-go/whisk"
	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

var (
	_       component.TracesProcessor = (*owTraceProcessor)(nil)
	retries int                       = 2
	sleep   time.Duration             = 5
)

type owTraceProcessor struct {
	next   consumer.TracesConsumer
	owclient *whisk.Client
	logger *zap.Logger
	logging bool
}

func newOwTraceProcessor(next consumer.TracesConsumer, cfg *Config, logger *zap.Logger) (*owTraceProcessor, error) {
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
	return &owTraceProcessor{next: next, owclient: c, logger: logger, logging: cfg.Logging}, nil
}

func (p *owTraceProcessor) ConsumeTraces(ctx context.Context, batch pdata.Traces) error {
	// ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]
	for i := 0; i < batch.ResourceSpans().Len(); i++ {
		rs := batch.ResourceSpans().At(i)
		for j := 0; j < rs.InstrumentationLibrarySpans().Len(); j++ {
			spans := rs.InstrumentationLibrarySpans().At(j).Spans()
			newSpans := pdata.NewSpanSlice()
			for k := 0; k < spans.Len(); k++ {
				executionSpan := spans.At(k)
				id, ok := executionSpan.Attributes().Get("activationId")
				if !ok && p.logging {
					p.logger.Info("Span " + executionSpan.SpanID().HexString() + " without activation id attribute")
				} else {
					if p.logging {
						p.logger.Info("Processing span " + executionSpan.SpanID().HexString() + " with activation id: " + id.StringVal())
					}
					activation, res, err := p.owclient.Activations.Get(id.StringVal())
					counter := retries
					for err != nil && counter > 0 {
						time.Sleep(sleep * time.Second)
						if p.logging {
							p.logger.Info("Access to OpenWhisk API failed for span " + executionSpan.SpanID().HexString() + " with activation id: " + id.StringVal() + ", status code: " + res.Status + ", error message: " + err.Error() + ". Retrying ...")
						}
						activation, res, err = p.owclient.Activations.Get(id.StringVal())
						counter--
					}
					if res.StatusCode == http.StatusOK {
						// OpenTelemetry works with nanoseconds, whereas the OpenWhisk API milliseconds returns
						executionStartNano := activation.Start * 1e06
						executionEndNano := activation.End * 1e06
						waitTime := activation.Annotations.GetValue("waitTime")
						initTime := activation.Annotations.GetValue("initTime")

						var waitTimeNano int64
						if waitTime != nil {
							waitTimeNano, _ = waitTime.(json.Number).Int64()
							waitTimeNano *= 1e06
							if p.logging {
								p.logger.Info("Span " + executionSpan.SpanID().HexString() + " with waitTime: " + strconv.FormatInt(waitTimeNano, 10) + " ns")
							}
						}
						var initTimeNano int64
						if initTime != nil {
							initTimeNano, _ = initTime.(json.Number).Int64()
							initTimeNano *= 1e6
							if p.logging {
								p.logger.Info("Span " + executionSpan.SpanID().HexString() + " with initTime: " + strconv.FormatInt(initTimeNano, 10) + " ns")
							}
						}
						// Create new parent span
						newParentSpan := pdata.NewSpan()
						executionSpan.CopyTo(newParentSpan)
						newParentSpan.SetSpanID(pdata.NewSpanID(createSpanID()))
						newParentSpan.SetStartTime(pdata.TimestampUnixNano(executionStartNano - waitTimeNano - initTimeNano))
						newParentSpan.SetEndTime(pdata.TimestampUnixNano(executionEndNano))
						newSpans.Append(newParentSpan)
						// Add execution span to new parent span
						executionSpan.SetParentSpanID(newParentSpan.SpanID())
						executionSpan.SetStartTime(pdata.TimestampUnixNano(executionStartNano))
						executionSpan.SetEndTime(pdata.TimestampUnixNano(executionEndNano))
						executionSpanName := executionSpan.Name()
						executionSpan.SetName("execution: " + executionSpanName)
						// If wait time present: create wait span and add to new parent span
						if waitTime != nil {
							waitSpan := pdata.NewSpan()
							executionSpan.CopyTo(waitSpan)
							waitSpan.SetSpanID(pdata.NewSpanID(createSpanID()))
							waitSpan.SetParentSpanID(newParentSpan.SpanID())
							waitSpan.SetName("waitTime: " + executionSpanName)
							waitSpan.SetStartTime(pdata.TimestampUnixNano(executionStartNano - initTimeNano - waitTimeNano))
							waitSpan.SetEndTime(pdata.TimestampUnixNano(executionStartNano - initTimeNano))
							newSpans.Append(waitSpan)
						}
						// If init time present: create init span and add to new parent span
						if initTime != nil {
							initSpan := pdata.NewSpan()
							executionSpan.CopyTo(initSpan)
							initSpan.SetSpanID(pdata.NewSpanID(createSpanID()))
							initSpan.SetParentSpanID(newParentSpan.SpanID())
							initSpan.SetName("initTime: " + executionSpanName)
							initSpan.SetStartTime(pdata.TimestampUnixNano(executionStartNano - initTimeNano))
							initSpan.SetEndTime(pdata.TimestampUnixNano(executionStartNano))
							newSpans.Append(initSpan)
						}
					} else {
						if p.logging {
							p.logger.Info("Unable to access OpenWhisk API for span " + executionSpan.SpanID().HexString() + " with activation id: " + id.StringVal() + ", status code: " + res.Status + ", error message: " + err.Error())
						}
					}
				}
			}
			newSpans.MoveAndAppendTo(spans)
		}
	}
	return p.next.ConsumeTraces(ctx, batch)
}

func createSpanID() [8]byte {
	uuid, _ := uuid.New().MarshalBinary()
	var result [8]byte
	copy(result[:], uuid[:8])
	return result
}

func (p *owTraceProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (p *owTraceProcessor) Start(_ context.Context, host component.Host) error {
	return nil
}

func (p *owTraceProcessor) Shutdown(context.Context) error {
	return nil
}
