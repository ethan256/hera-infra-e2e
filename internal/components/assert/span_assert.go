package assert

import (
	"os"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert/entity"
)

type HandlerFunc func(*SpanAssertContext) error

type HandlersChain []HandlerFunc

type SpanAssertContext struct {
	actual   *entity.Span
	expected *entity.Span

	handlers HandlersChain
}

type SpanAssertProvider struct {
	pool sync.Pool
}

var globalSpanAssertProvider = initSpanAssertProvider()

func initSpanAssertProvider() *SpanAssertProvider {
	p := &SpanAssertProvider{}
	p.pool.New = func() any {
		return allocateSpanAssertContext()
	}
	return p
}

func GetSpanAssert() *SpanAssertContext {
	return globalSpanAssertProvider.pool.Get().(*SpanAssertContext)
}

func allocateSpanAssertContext() *SpanAssertContext {
	return &SpanAssertContext{
		handlers: make(HandlersChain, 0),
	}
}

func (sac *SpanAssertContext) SetActualSpan(actual *entity.Span) *SpanAssertContext {
	sac.actual = actual
	return sac
}

func (sac *SpanAssertContext) SetExpectedSpan(expected *entity.Span) *SpanAssertContext {
	sac.expected = expected
	return sac
}

func (sac *SpanAssertContext) reset() {
	sac.handlers = sac.handlers[:0]
	sac.actual = nil
	sac.expected = nil
}

func (sac *SpanAssertContext) assert() error {
	defer func() {
		sac.reset()
		globalSpanAssertProvider.pool.Put(sac)
	}()

	for _, handler := range sac.handlers {
		if err := handler(sac); err != nil {
			return err
		}
	}
	return nil
}

func (sac *SpanAssertContext) assertSpanID() *SpanAssertContext {
	f := func(sac *SpanAssertContext) error {
		return ValueAssert("SpanID", sac.expected.SpanID, sac.actual.SpanID)
	}
	sac.handlers = append(sac.handlers, f)
	return sac
}

func (sac *SpanAssertContext) assertDuration() *SpanAssertContext {
	f := func(sac *SpanAssertContext) error {
		return ValueAssert("Duration", sac.expected.Duration.String(), sac.actual.Duration.String())
	}
	sac.handlers = append(sac.handlers, f)
	return sac
}

func (sac *SpanAssertContext) assertFlags() *SpanAssertContext {
	f := func(sac *SpanAssertContext) error {
		return ValueAssert("Flags", sac.expected.Flags.String(), sac.actual.Flags.String())
	}
	sac.handlers = append(sac.handlers, f)
	return sac
}

func (sac *SpanAssertContext) assertOperationName() *SpanAssertContext {
	f := func(sac *SpanAssertContext) error {
		return ValueAssert("OperationName", sac.expected.OperationName, sac.actual.OperationName)
	}
	sac.handlers = append(sac.handlers, f)
	return sac
}

func (sac *SpanAssertContext) assertStartTime() *SpanAssertContext {
	f := func(sac *SpanAssertContext) error {
		return ValueAssert("StartTime", sac.expected.StartTime.String(), sac.actual.StartTime.String())
	}
	sac.handlers = append(sac.handlers, f)
	return sac
}

func (sac *SpanAssertContext) assertParentSpanID() *SpanAssertContext {
	f := func(sac *SpanAssertContext) error {
		return ValueAssert("ParentSpanID", sac.expected.ParentSpanID, sac.actual.ParentSpanID)
	}
	sac.handlers = append(sac.handlers, f)
	return sac
}

func (sac *SpanAssertContext) assertLogs() *SpanAssertContext {
	// TODO 目前logs为空
	return sac
}

func (sac *SpanAssertContext) assertTags() *SpanAssertContext {
	f := func(sac *SpanAssertContext) error {
		return tagsEquals(sac.expected.Tags, sac.actual.Tags)
	}
	sac.handlers = append(sac.handlers, f)
	return sac
}

func (sac *SpanAssertContext) assertReferences() *SpanAssertContext {
	f := func(sac *SpanAssertContext) error {
		return refEquals(sac.expected.References, sac.actual.References)
	}
	sac.handlers = append(sac.handlers, f)
	return sac
}

func tagsEquals(expected, actual map[string]string) error {
	if len(expected) != len(actual) {
		return errors.Errorf("Span Tags Size Not Equal: assister=>%d, actual=>%d", len(expected), len(actual))
	}
	for k, v := range expected {
		if k == "http.url" {
			// there can be env variables in http.url, say, "http://${GATEWAY_HOST}:${GATEWAY_PORT}/test"
			v = os.ExpandEnv(v)
		}
		if err := ValueAssert(k, v, actual[k]); err != nil {
			return err
		}
	}
	return nil
}

func refEquals(excepted, actual []*entity.Reference) error {
	if excepted == nil {
		return nil
	}

	if actual == nil {
		return errors.New("Actual Data Reference is Empty")
	}

	if len(excepted) != len(actual) {
		return errors.Errorf("Reference Size not Equal: assister=>%d, actual=>%d", len(excepted), len(actual))
	}

	for _, ref := range excepted {
		if _, err := findReference(ref, actual); err != nil {
			return err
		}
	}
	return nil
}

func findReference(expected *entity.Reference, actual []*entity.Reference) (*entity.Reference, error) {
	var err error
	for _, segmentRef := range actual {
		if interErr := simpleReferenceEquals(expected, segmentRef); interErr != nil {
			err = multierr.Append(err, interErr)
		} else {
			return segmentRef, nil
		}
	}
	return nil, errors.Wrap(err, "ReferenceNotFoundException")
}

func simpleReferenceEquals(expected, actual *entity.Reference) error {
	if err := ValueAssert("reference type", expected.RefType, actual.RefType); err != nil {
		return err
	}
	if err := ValueAssert("span id", expected.SpanID, actual.SpanID); err != nil {
		return err
	}
	if err := ValueAssert("trace id", expected.TraceID, actual.TraceID); err != nil {
		return err
	}
	return nil
}
