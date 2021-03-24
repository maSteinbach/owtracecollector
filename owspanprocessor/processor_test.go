package owspanprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

func TestAddWaitAndInitTimeAttributes(t *testing.T) {

	// Traces[] -> ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]
	rs := pdata.NewResourceSpans()
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(2)
	
	span1 := ils.Spans().At(0)
	span1.SetName("first-execution-span")
	span1.SetTraceID(pdata.NewTraceID([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	// Cold activation -> results waitTime and initTime attribute.
	span1.Attributes().InsertString("activationId", "926ce30faca447e5ace30faca447e548") // Configure activationId according to your needs.
	
	span2 := ils.Spans().At(1)
	span2.SetName("second-execution-span")
	span2.SetTraceID(pdata.NewTraceID([16]byte{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	// Warm activation -> results in only waitTime attribute.
	span2.Attributes().InsertString("activationId", "4bec111e4f104864ac111e4f10f8643e") // Configure activationId according to your needs.

	inBatch := pdata.NewTraces()
	inBatch.ResourceSpans().Append(rs)

	// Test
	next := &componenttest.ExampleExporterConsumer{}
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	// Configure config according to your needs.
	config := &Config{
		OwHost:      "10.195.5.240:31001",
		OwAuthToken: "23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP",
		Logging:     true,
	}
	processor, err := newOwSpanProcessor(next, config, creationParams.Logger)
	processor.ConsumeTraces(context.Background(), inBatch)
	s1waitTimeNano, _ := next.Traces[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).Attributes().Get("waitTimeNano")
	s1initTimeNano, _ := next.Traces[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0).Attributes().Get("initTimeNano")
	s2waitTimeNano, _ := next.Traces[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(1).Attributes().Get("waitTimeNano")
	_, s2initTimePresent := next.Traces[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(1).Attributes().Get("initTimeNano")
	
	// Verify
	assert.NoError(t, err)
	assert.Equal(t, s1waitTimeNano.IntVal(), int64(4276000000))
	assert.Equal(t, s1initTimeNano.IntVal(), int64(128000000))
	assert.Equal(t, s2waitTimeNano.IntVal(), int64(18000000))
	assert.False(t, s2initTimePresent)
}