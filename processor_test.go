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

// TODO: do not hardcode activation id

func TestAddWaitAndInitSpan(t *testing.T) {

	// Traces[] -> ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]

	rs := pdata.NewResourceSpans()
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	//library := ils.InstrumentationLibrary()
	//library.InitEmpty()
	//library.SetName("first-library")
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetName("first-execution-span")
	span.SetTraceID(pdata.NewTraceID([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	// cold activation -> results in total 5 spans (parent, execution, wait, init)
	span.Attributes().InsertString("activationId", "926ce30faca447e5ace30faca447e548")

	rs2 := pdata.NewResourceSpans()
	rs2.InstrumentationLibrarySpans().Resize(1)
	ils2 := rs2.InstrumentationLibrarySpans().At(0)
	ils2.Spans().Resize(1)
	secondSpan := ils2.Spans().At(0)
	secondSpan.SetName("second-execution-span")
	secondSpan.SetTraceID(pdata.NewTraceID([16]byte{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	// warm activation -> results in total 4 spans (parent, execution, wait)
	secondSpan.Attributes().InsertString("activationId", "4bec111e4f104864ac111e4f10f8643e")
	
	inBatch := pdata.NewTraces()
	inBatch.ResourceSpans().Append(rs)
	inBatch.ResourceSpans().Append(rs2)

	// test
	next := &componenttest.ExampleExporterConsumer{}
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	processor := newOwTraceProcessor(next, creationParams.Logger)
	err := processor.ConsumeTraces(context.Background(), inBatch)

	// verify
	assert.NoError(t, err)
	//assert.Len(t, next.Traces, 2)
	assert.Equal(t, next.Traces[0].SpanCount(), 7);

	// first batch
	//firstOutILS := next.Traces[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	//assert.Equal(t, library.Name(), firstOutILS.InstrumentationLibrary().Name())
	//assert.Equal(t, firstSpan.Name(), firstOutILS.Spans().At(0).Name())

	// second batch
	//secondOutILS := next.Traces[1].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	//assert.Equal(t, library.Name(), secondOutILS.InstrumentationLibrary().Name())
	//assert.Equal(t, secondSpan.Name(), secondOutILS.Spans().At(0).Name())
}