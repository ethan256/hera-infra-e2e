package exception

import (
	"fmt"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert/entity"
)

type SpanAssertFailedError struct {
	expectedSpan *entity.Span
	actualSpan   *entity.Span
	message      string
}

func NewSpanAssertFailedError(expectedSpan, actualSpan *entity.Span, message string) *SpanAssertFailedError {
	return &SpanAssertFailedError{
		expectedSpan: expectedSpan,
		actualSpan:   actualSpan,
		message:      message,
	}
}

func (s *SpanAssertFailedError) getMessage() string {
	return s.message
}

func (s *SpanAssertFailedError) getExpectedSpan() *entity.Span {
	return s.expectedSpan
}

func (s *SpanAssertFailedError) getActualSpan() *entity.Span {
	return s.actualSpan
}

func (s *SpanAssertFailedError) Error() string {
	return fmt.Sprintf("expected:\tSpan[%s, %s] %s\n"+
		"actual:\tSpan[%s, %s] %s\n  "+
		"reason:\t%s\n",
		s.expectedSpan.ParentSpanID, s.expectedSpan.SpanID, s.expectedSpan.OperationName,
		s.actualSpan.ParentSpanID, s.actualSpan.SpanID, s.actualSpan.OperationName,
		s.message,
	)
}
