package owtraceprocessor

import (
	"context"
	"encoding/json"
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
	_       component.TracesProcessor = (*owTraceProcessor)(nil)
	retries int                       = 2
	sleep   time.Duration             = 5
)

type owTraceProcessor struct {
	next   consumer.TracesConsumer
	logger *zap.Logger
}

func newOwTraceProcessor(next consumer.TracesConsumer, logger *zap.Logger) *owTraceProcessor {
	return &owTraceProcessor{next: next, logger: logger}
}

func (p *owTraceProcessor) ConsumeTraces(ctx context.Context, batch pdata.Traces) error {

	config := &whisk.Config{
		Host:      "10.195.5.240:31001",
		AuthToken: "23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP",
		Insecure:  true,
	}
	client, err := whisk.NewClient(http.DefaultClient, config)
	if err != nil {
		p.logger.Info("Unable to connect to OpenWhisk \nError message: " + err.Error())
	} else {
		// ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]
		for i := 0; i < batch.ResourceSpans().Len(); i++ {
			rs := batch.ResourceSpans().At(i)
			for j := 0; j < rs.InstrumentationLibrarySpans().Len(); j++ {
				spans := rs.InstrumentationLibrarySpans().At(j).Spans()

				newSpans := pdata.NewSpanSlice()
				for k := 0; k < spans.Len(); k++ {
					executionSpan := spans.At(k)
					id, ok := executionSpan.Attributes().Get("activationId")
					if !ok {
						p.logger.Info("Span " + executionSpan.SpanID().HexString() + " without activation id attribute")
					} else {
						p.logger.Info("Processing span " + executionSpan.SpanID().HexString() + " with activation id: " + id.StringVal())
						activation, res, err := client.Activations.Get(id.StringVal())
						counter := retries
						for err != nil && counter > 0 {
							time.Sleep(sleep * time.Second)
							p.logger.Info("Access to OpenWhisk API failed for span " + executionSpan.SpanID().HexString() + " with activation id: " + id.StringVal() + ", status code: " + res.Status + ", error message: " + err.Error() + ". Retrying ...")
							activation, res, err = client.Activations.Get(id.StringVal())
							counter--
						}
						if res.StatusCode == http.StatusOK {
							// OpenTelemetry works with nanoseconds, whereas the OpenWhisk API milliseconds returns
							executionStartNano := activation.Start * 1e06
							waitTime := activation.Annotations.GetValue("waitTime")
							initTime := activation.Annotations.GetValue("initTime")

							var waitTimeNano int64
							if waitTime != nil {
								waitTimeNano, _ = waitTime.(json.Number).Int64()
								waitTimeNano *= 1e06
								p.logger.Info("Span " + executionSpan.SpanID().HexString() + " with waitTime: " + strconv.FormatInt(waitTimeNano, 10) + " ns")
							}
							var initTimeNano int64
							if initTime != nil {
								initTimeNano, _ = initTime.(json.Number).Int64()
								initTimeNano *= 1e6
								p.logger.Info("Span " + executionSpan.SpanID().HexString() + " with initTime: " + strconv.FormatInt(initTimeNano, 10) + " ns")
							}

							// create new parent span
							newParentSpan := pdata.NewSpan()
							executionSpan.CopyTo(newParentSpan)
							newParentSpan.SetSpanID(executionSpan.SpanID())
							newParentSpan.SetStartTime(pdata.TimestampUnixNano(executionStartNano - waitTimeNano - initTimeNano))
							newSpans.Append(newParentSpan)

							// add execution span to new parent span
							executionSpan.SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
							executionSpan.SetParentSpanID(newParentSpan.SpanID())
							executionSpanName := executionSpan.Name()
							executionSpan.SetName("execution: " + executionSpanName)

							// if wait time present: create wait span and add to new parent span
							if waitTime != nil {
								waitSpan := pdata.NewSpan()
								executionSpan.CopyTo(waitSpan)
								waitSpan.SetSpanID(pdata.NewSpanID([8]byte{2, 3, 4, 5, 6, 7, 8, 9}))
								waitSpan.SetParentSpanID(newParentSpan.SpanID())
								waitSpan.SetName("waitTime: " + executionSpanName)
								waitSpan.SetStartTime(pdata.TimestampUnixNano(executionStartNano - initTimeNano - waitTimeNano))
								waitSpan.SetEndTime(pdata.TimestampUnixNano(executionStartNano - initTimeNano))
								newSpans.Append(waitSpan)
							}

							// if init time present: create init span and add to new parent span
							if initTime != nil {
								initSpan := pdata.NewSpan()
								executionSpan.CopyTo(initSpan)
								initSpan.SetSpanID(pdata.NewSpanID([8]byte{3, 4, 5, 6, 7, 8, 9, 0}))
								initSpan.SetParentSpanID(newParentSpan.SpanID())
								initSpan.SetName("initTime: " + executionSpanName)
								initSpan.SetStartTime(pdata.TimestampUnixNano(executionStartNano - initTimeNano))
								initSpan.SetEndTime(pdata.TimestampUnixNano(executionStartNano))
								newSpans.Append(initSpan)
							}
						} else {
							p.logger.Info("Unable to access OpenWhisk API for span " + executionSpan.SpanID().HexString() + " with activation id: " + id.StringVal() + ", status code: " + res.Status + ", error message: " + err.Error())
						}
					}
				}
				newSpans.MoveAndAppendTo(spans)
			}
		}
	}
	if err := p.next.ConsumeTraces(ctx, batch); err != nil {
		return err
	}
	return nil
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
