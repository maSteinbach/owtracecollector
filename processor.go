package owtraceprocessor

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/apache/openwhisk-client-go/whisk"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var (
	_ component.TracesProcessor = (*owTraceProcessor)(nil)
	retries int = 2
	sleep time.Duration = 5
)

type owTraceProcessor struct {
	next consumer.TracesConsumer
}

func newOwTraceProcessor(next consumer.TracesConsumer) *owTraceProcessor {
	return &owTraceProcessor{next: next}
}

func (s *owTraceProcessor) ConsumeTraces(ctx context.Context, batch pdata.Traces) error {
	
	config := &whisk.Config{
		Host: "10.195.5.240:31001",
		AuthToken: "23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP",
		Insecure: true,
	}
	client, err := whisk.NewClient(http.DefaultClient, config)
	if err != nil {
		log.Println(err)
	} else {
		// ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]
		for i := 0; i < batch.ResourceSpans().Len(); i++ {
			rs := batch.ResourceSpans().At(i);
			for j := 0; j < rs.InstrumentationLibrarySpans().Len(); j++ {
				spans := rs.InstrumentationLibrarySpans().At(j).Spans();
				
				newSpans := pdata.NewSpanSlice()
				for k := 0; k < spans.Len(); k++ {
					executionSpan := spans.At(k)
					id, ok := executionSpan.Attributes().Get("activationId")
					if !ok {
						log.Println("activation id not found")
					} else {
						activation, res, err := client.Activations.Get(id.StringVal())
						for err != nil && retries > 0 {
							time.Sleep(sleep * time.Second)
							activation, res, err = client.Activations.Get(id.StringVal())
							retries--
						}
						if res.StatusCode == http.StatusOK {
							executionStart := activation.Start
							waitTime := activation.Annotations.GetValue("waitTime")
							initTime := activation.Annotations.GetValue("initTime")

							var waitTimeVal int64
							if waitTime != nil {
								waitTimeVal, _ = waitTime.(json.Number).Int64()
							}
							var initTimeVal int64
							if initTime != nil {
								initTimeVal, _ = initTime.(json.Number).Int64()
							}

							// create new parent span
							newParentSpan := pdata.NewSpan()
							executionSpan.CopyTo(newParentSpan)
							newParentSpan.SetSpanID(executionSpan.SpanID())
							newParentSpan.SetStartTime(pdata.TimestampUnixNano(executionStart - waitTimeVal - initTimeVal))
							newSpans.Append(newParentSpan)
		
							// add execution span to new parent span
							executionSpan.SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
							executionSpan.SetParentSpanID(newParentSpan.SpanID())
							executionSpan.SetName("execution: " + executionSpan.Name())

							// if wait time present: create wait span and add to new parent span
							if waitTime != nil {
								waitSpan := pdata.NewSpan()
								executionSpan.CopyTo(waitSpan)
								waitSpan.SetSpanID(pdata.NewSpanID([8]byte{2, 3, 4, 5, 6, 7, 8, 9}))
								waitSpan.SetParentSpanID(newParentSpan.SpanID())
								waitSpan.SetName("waitTime: " + waitSpan.Name())
								waitSpan.SetStartTime(pdata.TimestampUnixNano(executionStart - initTimeVal - waitTimeVal))
								waitSpan.SetEndTime(pdata.TimestampUnixNano(executionStart - initTimeVal))
								newSpans.Append(waitSpan)
							}
							
							// if init time present: create init span and add to new parent span
							if initTime != nil {
								initSpan := pdata.NewSpan()
								executionSpan.CopyTo(initSpan)
								initSpan.SetSpanID(pdata.NewSpanID([8]byte{3, 4, 5, 6, 7, 8, 9, 0}))
								initSpan.SetParentSpanID(newParentSpan.SpanID())
								initSpan.SetName("initTime: " + initSpan.Name())
								initSpan.SetStartTime(pdata.TimestampUnixNano(executionStart - initTimeVal))
								initSpan.SetEndTime(pdata.TimestampUnixNano(executionStart))
								newSpans.Append(initSpan)
							}
						}
					}
				}
				newSpans.MoveAndAppendTo(spans)
			}
		}
	}
	if err := s.next.ConsumeTraces(ctx, batch); err != nil {
		return err
	}
	return nil
}

func (s *owTraceProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (s *owTraceProcessor) Start(_ context.Context, host component.Host) error {
	return nil
}

func (s *owTraceProcessor) Shutdown(context.Context) error {
	return nil
}