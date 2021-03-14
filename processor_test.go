package owtraceprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

func TestAddWaitAndInitSpan(t *testing.T) {

	// Traces[] -> ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]
	rs := pdata.NewResourceSpans()
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetName("first-execution-span")
	span.SetTraceID(pdata.NewTraceID([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	// Cold activation -> results in total 5 spans (parent, execution, wait, init)
	span.Attributes().InsertString("activationId", "926ce30faca447e5ace30faca447e548")

	rs2 := pdata.NewResourceSpans()
	rs2.InstrumentationLibrarySpans().Resize(1)
	ils2 := rs2.InstrumentationLibrarySpans().At(0)
	ils2.Spans().Resize(1)
	secondSpan := ils2.Spans().At(0)
	secondSpan.SetName("second-execution-span")
	secondSpan.SetTraceID(pdata.NewTraceID([16]byte{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	// Warm activation -> results in total 4 spans (parent, execution, wait)
	secondSpan.Attributes().InsertString("activationId", "4bec111e4f104864ac111e4f10f8643e")
	
	inBatch := pdata.NewTraces()
	inBatch.ResourceSpans().Append(rs)
	inBatch.ResourceSpans().Append(rs2)

	// Test
	next := &componenttest.ExampleExporterConsumer{}
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	config := &Config{
		OwHost: "10.195.5.240:31001",
		OwAuthToken: "23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP",
	}
	processor, err := newOwTraceProcessor(next, config, creationParams.Logger)
	processor.ConsumeTraces(context.Background(), inBatch)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, next.Traces[0].SpanCount(), 7);
}