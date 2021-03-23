package owspanattacher

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

var (
	_       component.TracesProcessor = (*owSpanAttacher)(nil)
)

type owSpanAttacher struct {
	next     consumer.TracesConsumer
	logger   *zap.Logger
	logging  bool
}

func newOwSpanAttacher(next consumer.TracesConsumer, cfg *Config, logger *zap.Logger) (*owSpanAttacher, error) {
	return &owSpanAttacher{next: next, logger: logger, logging: cfg.Logging}, nil
}

func (p *owSpanAttacher) ConsumeTraces(ctx context.Context, batch pdata.Traces) error {
	// ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]
	for i := 0; i < batch.ResourceSpans().Len(); i++ {
		rs := batch.ResourceSpans().At(i)
		for j := 0; j < rs.InstrumentationLibrarySpans().Len(); j++ {
			spans := rs.InstrumentationLibrarySpans().At(j).Spans()
			newSpans := pdata.NewSpanSlice()
			for k := 0; k < spans.Len(); k++ {
				parentSpan := spans.At(k)
				// Get optional attributes waitTime and initTime from span.
				var waitTimeNano pdata.TimestampUnixNano
				waitTime, ok := parentSpan.Attributes().Get("waitTimeNano")
				if ok {
					waitTimeNano = pdata.TimestampUnixNano(waitTime.IntVal())
				} 
				var initTimeNano pdata.TimestampUnixNano
				initTime, ok := parentSpan.Attributes().Get("initTimeNano")
				if ok {
					initTimeNano = pdata.TimestampUnixNano(initTime.IntVal())
				}
				if p.logging {
					p.logger.Info("Attaching wait, init and execution spans to span " + parentSpan.SpanID().HexString())
				}
				// Create execution span.
				newSpans.Append(
					p.createSpan(
						"Execution: " + parentSpan.Name(),
						parentSpan.TraceID(),
						parentSpan.SpanID(),
						parentSpan.StartTime() + waitTimeNano + initTimeNano,
						parentSpan.EndTime(),
					),
				)
				// Create wait span.
				if waitTimeNano > 0 {
					newSpans.Append(
						p.createSpan(
							"Wait: " + parentSpan.Name(),
							parentSpan.TraceID(),
							parentSpan.SpanID(),
							parentSpan.StartTime(),
							parentSpan.StartTime() + waitTimeNano,
						),
					)
				}
				// Create init span.
				if initTimeNano > 0 {
					newSpans.Append(
						p.createSpan(
							"Init: " + parentSpan.Name(),
							parentSpan.TraceID(),
							parentSpan.SpanID(),
							parentSpan.StartTime() + waitTimeNano,
							parentSpan.StartTime() + waitTimeNano + initTimeNano,
						),
					)
				}
			}
			newSpans.MoveAndAppendTo(spans)
		}
	}
	return p.next.ConsumeTraces(ctx, batch)
}

func (p *owSpanAttacher) createSpan(
	name string, 
	traceId pdata.TraceID, 
	parentSpanId pdata.SpanID, 
	startTime pdata.TimestampUnixNano, 
	endTime pdata.TimestampUnixNano,
) pdata.Span {
	span := pdata.NewSpan()
	span.SetName(name)
	span.SetTraceID(traceId)
	span.SetSpanID(pdata.NewSpanID(p.createSpanID()))
	span.SetParentSpanID(parentSpanId)
	span.SetStartTime(startTime)
	span.SetEndTime(endTime)
	return span
}

func (p *owSpanAttacher) createSpanID() [8]byte {
	uuid, _ := uuid.New().MarshalBinary()
	var result [8]byte
	copy(result[:], uuid[:8])
	return result
}

func (p *owSpanAttacher) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (p *owSpanAttacher) Start(_ context.Context, host component.Host) error {
	return nil
}

func (p *owSpanAttacher) Shutdown(context.Context) error {
	return nil
}