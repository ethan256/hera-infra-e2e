package assert

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/prometheus/prom2json"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {
	cases := []struct {
		name             string
		expectedDataFile string
		actualMetrics    []*prom2json.Family
		err              string
	}{
		{
			name:             "MySQL Scenario - Empty labels",
			expectedDataFile: "./testdata/mysqlExpectedMetrics.json",
			actualMetrics: []*prom2json.Family{
				{
					Name: "jdbc_operations:aggr_seconds",
					Type: "SUMMARY",
					Metrics: []interface{}{
						prom2json.Summary{
							TimestampMs: "1000000",
							Count:       "1",
							Sum:         "1000",
							Quantiles: map[string]string{
								"0.5":  "0.1",
								"0.75": "0.2",
								"0.95": "0.3",
								"0.99": "0.4",
							},
						},
					},
				},
			},
			err: "Metric[jdbc_operations:aggr_seconds]",
		},
		{
			name:             "MySQL Scenario - More labels but not included",
			expectedDataFile: "./testdata/mysqlExpectedMetrics.json",
			actualMetrics: []*prom2json.Family{
				{
					Name: "jdbc_operations:aggr_seconds",
					Type: "SUMMARY",
					Metrics: []interface{}{
						prom2json.Summary{
							TimestampMs: "1000000",
							Count:       "1",
							Sum:         "1000",
							Quantiles: map[string]string{
								"0.5":  "0.1",
								"0.75": "0.2",
								"0.95": "0.3",
								"0.99": "0.4",
							},
							Labels: map[string]string{
								"k1": "v1",
								"k2": "v2",
								"k3": "v3",
							},
						},
					},
				},
			},
			err: "Metric[jdbc_operations:aggr_seconds]",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := require.New(t)
			content, err := os.ReadFile(tt.expectedDataFile)
			assert.NoError(err)
			var expected []*prom2json.Family
			assert.NoError(json.Unmarshal(content, &expected))
			err = doMetricsAssert(expected, tt.actualMetrics)
			if tt.err == "" {
				assert.NoError(err)
			} else {
				assert.Error(err)
				assert.Contains(err.Error(), tt.err)
			}
		})
	}
}
