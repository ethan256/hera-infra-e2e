package assert

import (
	"testing"

	"bou.ke/monkey"
	tassert "github.com/stretchr/testify/assert"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert"
)

func TestAssertSingleCase(t *testing.T) {
	monkey.Patch(assert.MetricsAssert, func(string, string) error {
		panic("assert throws a panic")
	})
	err := assertSingleCase("", "", "")
	tassert.NotNil(t, err)
}
