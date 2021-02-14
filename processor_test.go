package owtraceprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestAddWaitAndInitSpan(t *testing.T) {

	// Traces[] -> ResourceSpans[] -> InstrumentationLibrarySpans[] -> Spans[]

	rs := pdata.NewResourceSpans()
	//rs.InitEmpty()
	rs.InstrumentationLibrarySpans().Resize(1)

	ils := rs.InstrumentationLibrarySpans().At(0)
	library := ils.InstrumentationLibrary()
	//library.InitEmpty()
	library.SetName("first-library")
	//ils.Spans().Resize(2)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetName("execution-span")
	span.SetTraceID(pdata.NewTraceID([16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}))
	span.Attributes().InsertString("activationId", "926ce30faca447e5ace30faca447e548")
	//secondSpan := ils.Spans().At(1)
	//secondSpan.SetName("first-batch-second-span")
	//secondSpan.SetTraceID(pdata.NewTraceID([]byte{2, 3, 4, 5}))

	inBatch := pdata.NewTraces()
	inBatch.ResourceSpans().Append(rs)

	// test
	next := &componenttest.ExampleExporterConsumer{}
	processor := newOwTraceProcessor(next)
	err := processor.ConsumeTraces(context.Background(), inBatch)

	// verify
	assert.NoError(t, err)
	//assert.Len(t, next.Traces, 2)
	assert.Len(t, next.Traces[0].SpanCount(), 3);

	// first batch
	//firstOutILS := next.Traces[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	//assert.Equal(t, library.Name(), firstOutILS.InstrumentationLibrary().Name())
	//assert.Equal(t, firstSpan.Name(), firstOutILS.Spans().At(0).Name())

	// second batch
	//secondOutILS := next.Traces[1].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0)
	//assert.Equal(t, library.Name(), secondOutILS.InstrumentationLibrary().Name())
	//assert.Equal(t, secondSpan.Name(), secondOutILS.Spans().At(0).Name())
}