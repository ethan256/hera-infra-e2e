package assert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	expect_data_path = "../../../test/assert/expectedData.json"
	actual_data_path = "../../../test/assert/actualData.json"
)

func TestLoadTracesData(t *testing.T) {
	tracesData, err := LoadTracesData(expect_data_path)
	if err != nil {
		t.Fatal(err)
	}

	if assert.NotEmpty(t, tracesData.Traces) {
		assert.Greater(t, len(tracesData.Traces), 0)
		assert.Greater(t, len(tracesData.Traces[0].Spans), 0)
		assert.NotEmpty(t, tracesData.Traces[0].Spans[0].OperationName)
	}
}

func TestDataAssert(t *testing.T) {
	err := DataAssert(expect_data_path, actual_data_path, "")
	assert.Equal(t, err, nil)
}
