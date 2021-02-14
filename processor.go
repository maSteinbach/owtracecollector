package owtraceprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var _ component.TracesProcessor = (*owTraceProcessor)(nil)

type owTraceProcessor struct {
	next consumer.TracesConsumer
}

func newOwTraceProcessor(next consumer.TracesConsumer) *owTraceProcessor {
	return &owTraceProcessor{next: next}
}

func (s *owTraceProcessor) ConsumeTraces(ctx context.Context, batch pdata.Traces) error {
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