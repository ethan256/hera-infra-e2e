package assert

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/prometheus/prom2json"
	"github.com/stretchr/testify/assert"
)

const (
	expect_data_path    = "../../../test/assert/expectedTraces.json"
	actual_data_path    = "../../../test/assert/actualTraces.json"
	expect_metrics_path = "../../../test/assert/expectedMetrics.json"
)

func TestLoadTracesData(t *testing.T) {
	tracesData, err := LoadTracesData(actual_data_path)
	if err != nil {
		t.Fatal(err)
	}

	if assert.NotEmpty(t, tracesData) {
		assert.Greater(t, tracesData.Size, 0)
		assert.Greater(t, len(tracesData.Traces[0].Spans), 0)
		assert.NotEmpty(t, tracesData.Traces[0].Spans[0].OperationName)
	}
}

func TestUmmarslMetrics(t *testing.T) {
	data, err := os.ReadFile(expect_metrics_path)
	assert.NoError(t, err)
	var metrics []*prom2json.Family
	err = json.Unmarshal(data, &metrics)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(metrics))
}
