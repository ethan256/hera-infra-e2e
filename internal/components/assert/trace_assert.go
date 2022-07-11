package assert

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert/entity"
	"github.com/apache/skywalking-infra-e2e/internal/components/assert/exception"
)

// doTracesAssert assert expected traces and actual traces.
// return nil, if assert success, otherwise， return error.
// the first matching trace is searched from the actual data
// until all expected traces are successfully matched.
// Once the search fails, return error directly
func doTracesAssert(expected, actual []*entity.Trace) error {
	if len(expected) == 0 {
		return errors.New("expected traces can not empty")
	}

	var actualTrace *entity.Trace
	var err error
	exist := make(map[string]struct{})
	for i := 0; i < len(expected); i++ {
		if actualTrace, err = findTrace(expected[i], actual, exist); err != nil {
			return err
		}
		expected[i].TraceID = actualTrace.TraceID
	}
	return nil
}

// findTrace find the first matching expected trace from the actual data.
func findTrace(expectedTrace *entity.Trace, actual []*entity.Trace, exist map[string]struct{}) (*entity.Trace, error) {
	var err error
	var actualTrace *entity.Trace

	for _, actualTrace = range actual {
		if _, ok := exist[actualTrace.TraceID]; !ok {
			if err = spansAssert(expectedTrace, actualTrace); err == nil {
				exist[actualTrace.TraceID] = struct{}{}
				return actualTrace, nil
			}
		}
	}

	if spanAssertFailedError, ok := err.(*exception.SpanAssertFailedError); ok {
		return nil, exception.NewTraceNotFoundError(expectedTrace, spanAssertFailedError)
	}
	return nil, errors.Wrapf(err, "TraceID[%s]:\n  reason:\t%s", actualTrace.TraceID, err.Error())
}

// spansAssert assert expected spans from expectedTrace and actual spans from actualTrace.
// assert all spans from a specific trace. return error if the size of expected spans is different from
// the size of actual spans, or not match a expected span from actual spans.
func spansAssert(expectedTrace, actualTrace *entity.Trace) error {
	size := len(expectedTrace.Spans)
	if size != len(actualTrace.Spans) {
		return fmt.Errorf("SpansSizeNotEqual: expected=>%d, actual=> %d", size, len(actualTrace.Spans))
	}

	exist := sync.Map{}
	wg := sync.WaitGroup{}
	errChan := make(chan error, size)
	for i := 0; i < size; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := findSpan(expectedTrace.Spans[i], actualTrace, &exist); err != nil {
				errChan <- err
				return
			}
		}(i)
	}
	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// findSpan find a matched span from actual spans
// first, find same operation name spans. second, assert spans from same operation name spans.
// return actual span if a actual span assert success, otherwise return nil and error.
func findSpan(expectedSpan *entity.Span, actualTrace *entity.Trace, exist *sync.Map) (*entity.Span, error) {
	var err error
	var actualSpan *entity.Span

	sameOpNameSpans := findSpansWithSameOperationName(expectedSpan, actualTrace)
	if len(sameOpNameSpans) == 0 {
		return nil, exception.NewSpanAssertFailedError(expectedSpan, actualTrace.Spans[0],
			fmt.Sprintf("--expectedValue => %s, ++actualValue => %s", expectedSpan.OperationName, actualTrace.Spans[0].OperationName),
		)
	}

	for _, actualSpan = range sameOpNameSpans {
		if _, ok := exist.Load(actualSpan.SpanID); !ok {
			if err = spanAssert(expectedSpan, actualSpan); err == nil {
				exist.Store(actualSpan.SpanID, struct{}{})
				return actualSpan, nil
			}
		}
	}
	return nil, exception.NewSpanAssertFailedError(expectedSpan, actualSpan, err.Error())
}

// findSpansWithSameOperationName find all spans of same operation name from actual spans of a actual trace.
func findSpansWithSameOperationName(expectedSpan *entity.Span, actualTrace *entity.Trace) []*entity.Span {
	res := make([]*entity.Span, 0)

	for i := 0; i < len(actualTrace.Spans); i++ {
		if err := allocateSpanAssertContext().SetExpectedSpan(expectedSpan).SetActualSpan(actualTrace.Spans[i]).assert(); err == nil {
			res = append(res, actualTrace.Spans[i])
		}
	}
	return res
}

// spanAssert assert span
func spanAssert(expectedSpan, actualSpan *entity.Span) error {
	return GetSpanAssert().
		SetExpectedSpan(expectedSpan).
		SetActualSpan(actualSpan).
		assertSpanID().
		assertDuration().
		assertFlags().
		assertOperationName().
		assertStartTime().
		assertParentSpanID().
		assertLogs().
		assertTags().
		assertReferences().
		assert()
}
