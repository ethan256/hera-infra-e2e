package assert

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"sort"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert/entity"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
)

// DataAssert assert expected and actual data. return nil, if assert success, else return err.
// expectedPath is expected data file path.
// actualPath is actual data file path.
// query is the URL of the metric interface.
func DataAssert(expectedPath, actualPath, query string) error {
	var actual, expected *entity.Data
	var err error

	if expected, err = LoadTracesData(expectedPath); err != nil {
		return err
	}
	if actual, err = LoadTracesData(actualPath); err != nil {
		return err
	}

	if expected.ServiceName != "" && expected.ServiceName != actual.ServiceName {
		return errors.Errorf("Data.ServiceName Not Equal: assister=>%s, actual=>%s", expected.ServiceName, actual.ServiceName)
	}
	// Assert Traces
	err = TracesAssert(expected.Traces, actual.Traces)
	if err != nil {
		return err
	}
	logger.Log.Info("Assert traces success")

	// Assert Metrics
	if query != "" {
		metricsData, err := LoadMetricsData(query)
		if err != nil {
			return err
		}
		err = MetricsAssert(expected.Metrics, metricsData)
		if err != nil {
			return err
		}
		logger.Log.Info("Assert metrics success")
	}
	return nil
}

// LoadTracesData load traces data from file `path`.
func LoadTracesData(path string) (*entity.Data, error) {
	var (
		output  entity.Data
		content string
		err     error
	)

	if path != "" {
		content, err = util.ReadFileContent(path)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("path and query is empty")
	}
	if err = json.Unmarshal([]byte(content), &output); err != nil {
		return nil, errors.Wrap(err, "json unmarshal error")
	}

	for idx := range output.Traces {
		sort.Slice(output.Traces[idx].Spans, func(i, j int) bool {
			return output.Traces[idx].Spans[i].StartTime.IntValue() < output.Traces[idx].Spans[j].StartTime.IntValue()
		})
	}

	return &output, nil
}

// LoadMetricsData load metrics by query metric path or read content from file path.
func LoadMetricsData(path string) ([]*prom2json.Family, error) {
	var input io.Reader
	var err error
	path = os.ExpandEnv(path)
	if parse, urlErr := url.Parse(path); urlErr != nil || parse.Scheme == "" {
		// `parse, err := parse.Parse("/some/path.txt")` results in: `err == nil && parse.Scheme == ""`
		// Open file since arg appears not to be a valid URL (parsing error occurred or the scheme is missing).
		if input, err = os.Open(path); err != nil {
			return nil, errors.Wrapf(err, "path not found")
		}
	}

	mfChan := make(chan *dto.MetricFamily, 1024)
	if input != nil {
		go func(in io.Reader) {
			if err := prom2json.ParseReader(in, mfChan); err != nil {
				logger.Log.Error(err)
			}
		}(input)
	} else {
		go func() {
			if err := prom2json.FetchMetricFamilies(path, mfChan, nil); err != nil {
				logger.Log.Error(err)
			}
		}()
	}

	var result []*prom2json.Family
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}
	return result, nil
}
