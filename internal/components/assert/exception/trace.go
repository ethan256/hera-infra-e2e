package exception

import (
	"fmt"
	"strings"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert/entity"
)

type TraceNotFoundError struct {
	expected     *entity.Trace
	failedCauses *SpanAssertFailedError
}

func NewTraceNotFoundError(expected *entity.Trace, failedCauses *SpanAssertFailedError) *TraceNotFoundError {
	return &TraceNotFoundError{
		expected:     expected,
		failedCauses: failedCauses,
	}
}

func (t *TraceNotFoundError) Error() string {
	builder := strings.Builder{}
	builder.WriteString("\n  Trace:\n")
	for i := 0; i < len(t.expected.Spans); i++ {
		builder.WriteString(fmt.Sprintf("  - Span[%s, %s] %s\n",
			t.expected.Spans[i].ParentSpanID, t.expected.Spans[i].SpanID, t.expected.Spans[i].OperationName),
		)
	}

	expectedMsg := builder.String()
	builder.Reset()

	actualSpan := t.failedCauses.getActualSpan()
	expectedSpan := t.failedCauses.getExpectedSpan()

	causeMessage := fmt.Sprintf("  \nTrace[%s]:\n"+
		"  - expected:\tSpan[%s, %s] %s\n"+
		"  + actual:\tSpan[%s, %s] %s\n reason:\t%s",
		actualSpan.TraceID, expectedSpan.ParentSpanID, expectedSpan.SpanID, expectedSpan.OperationName,
		actualSpan.ParentSpanID, actualSpan.SpanID, actualSpan.OperationName, t.failedCauses.getMessage(),
	)
	return fmt.Sprintf("TraceNotFoundError:\nexpected: %s\nactual: %s\n", expectedMsg, causeMessage)
}
