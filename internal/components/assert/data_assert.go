package assert

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	"golang.org/x/exp/slices"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert/entity"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// TracesAssert assert trace message with expected
func TracesAssert(expectedPath, actualPath string) error {
	var actual, expected *entity.TraceData
	var err error

	if expectedPath == "" || actualPath == "" {
		return errors.New("expectedPath or actualPath is empty")
	}

	logger.Log.Debugf("start assert traces, expectedPath: %s, actualPath: %s", expectedPath, actualPath)
	if expected, err = LoadTracesData(expectedPath); err != nil {
		return err
	}
	if actual, err = LoadTracesData(actualPath); err != nil {
		return err
	}

	// Assert Traces
	err = doTracesAssert(expected.Traces, actual.Traces)
	if err != nil {
		return err
	}
	logger.Log.Debug("Assert traces success")
	return nil
}

// LoadTracesData load traces data from file `path`.
func LoadTracesData(path string) (*entity.TraceData, error) {
	var (
		output  *entity.TraceData
		content string
		err     error
	)

	// read trace data from file
	if content, err = util.ReadFileContent(path); err != nil {
		return nil, errors.Wrapf(err, "failed to read trace from file: %s", path)
	}

	if err = json.Unmarshal([]byte(content), &output); err != nil {
		return nil, errors.Wrap(err, "json unmarshal error")
	}

	for idx := range output.Traces {
		slices.SortFunc(output.Traces[idx].Spans, func(a, b *entity.Span) bool {
			return a.StartTime.IntValue() < b.StartTime.IntValue()
		})
	}

	return output, nil
}

// AssertMetrics assert metrics message
func MetricsAssert(expectedPath, query string) error {
	var expected []*prom2json.Family
	var actual []*prom2json.Family
	var err error

	logger.Log.Debugf("start assert metrics, expectedPath: %s, query: %s", expectedPath, query)
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		return errors.Wrap(err, "failed to read expected metrics")
	}
	err = json.Unmarshal(data, &expected)
	if err != nil {
		return errors.Wrap(err, "failed unmarshal expected metrics")
	}

	if actual, err = LoadMetricsData(query); err != nil {
		return err
	}

	err = doMetricsAssert(expected, actual)
	if err != nil {
		return err
	}
	logger.Log.Debug("Assert metrics success")
	return nil
}

// LoadMetricsData load metrics from request url.
func LoadMetricsData(url string) ([]*prom2json.Family, error) {
	url = os.ExpandEnv(url)
	mfChan := make(chan *dto.MetricFamily, 1024)
	go func() {
		if err := prom2json.FetchMetricFamilies(url, mfChan, nil); err != nil {
			logger.Log.Errorf("failed to query metric data from url: %s, error: %v", url, err)
		}
	}()

	var result []*prom2json.Family
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}
	return result, nil
}
