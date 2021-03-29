package owspanattacher

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
	span.SetName("parentspan")
	span.SetTraceID(pdata.NewTraceID([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	span.SetSpanID(pdata.NewSpanID([8]byte{0, 1, 2, 3, 4, 5, 6, 7}))
	span.SetStartTime(pdata.TimestampUnixNano(1616439288861000000))
	span.SetEndTime(pdata.TimestampUnixNano(1616439289383000000))
	span.Attributes().InsertInt("waitTimeMilli", 5463)
	span.Attributes().InsertInt("initTimeMilli", 168)

	inBatch := pdata.NewTraces()
	inBatch.ResourceSpans().Append(rs)

	// Test
	next := &componenttest.ExampleExporterConsumer{}
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	config := &Config{
		Logging:     true,
	}
	processor, err := newOwSpanAttacher(next, config, creationParams.Logger)
	processor.ConsumeTraces(context.Background(), inBatch)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, next.Traces[0].SpanCount(), 4)
}
