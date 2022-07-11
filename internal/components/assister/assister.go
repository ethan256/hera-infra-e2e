package assister

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/prom2json"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert"
	"github.com/apache/skywalking-infra-e2e/internal/components/assert/entity"
)

const (
	NotEmpty = "notEmpty"
	GeZero   = "ge 0"
	GtZero   = "gt 0"
)

func ActualDataToExpected(actualPath, expectedPath, url string) error {
	if url != "" {
		return writeExpectedMetricsToFile(url, expectedPath)
	}
	return writeExpectedTracesToFile(actualPath, expectedPath)
}

func writeExpectedTracesToFile(actualPath, expectedPath string) error {
	var (
		actual   *entity.TraceData
		expected *entity.TraceData
		res      []byte
		err      error
	)
	actual, err = assert.LoadTracesData(actualPath)
	if err != nil {
		return err
	}

	expected = &entity.TraceData{
		Traces: make([]*entity.Trace, 0),
	}
	traces := convertTraces(actual.Traces)
	expected.Traces = append(expected.Traces, traces...)
	expected.Size = actual.Size

	res, err = json.Marshal(expected)
	if err != nil {
		return err
	}

	return os.WriteFile(expectedPath, res, os.ModePerm)
}

func convertTraces(actual []*entity.Trace) []*entity.Trace {
	expected := make([]*entity.Trace, 0)
	for idx := range actual {
		trace := entity.Trace{TraceID: NotEmpty}
		for jdx := range actual[idx].Spans {
			span := actual[idx].Spans[jdx]
			span.TraceID = NotEmpty
			span.SpanID = NotEmpty
			span.ParentSpanID = NotEmpty
			span.Duration = entity.FromString(GeZero)
			span.StartTime = entity.FromString(GtZero)
			for kdx := range span.References {
				span.References[kdx].TraceID = NotEmpty
				span.References[kdx].SpanID = NotEmpty
			}
			trace.Spans = append(trace.Spans, span)
		}
		expected = append(expected, &trace)
	}
	return expected
}

func writeExpectedMetricsToFile(url, expectedPath string) error {
	var (
		expected []*prom2json.Family
		actual   []*prom2json.Family
		res      []byte
		err      error
	)

	actual, err = assert.LoadMetricsData(url)
	if err != nil {
		return err
	}
	metrics, err := convertMetrics(actual)
	if err != nil {
		return err
	}
	expected = append(expected, metrics...)
	res, err = json.Marshal(expected)
	if err != nil {
		return err
	}

	return os.WriteFile(expectedPath, res, os.ModePerm)
}

func convertMetrics(actualData []*prom2json.Family) ([]*prom2json.Family, error) {
	var expectedData []*prom2json.Family
	var err error
	for _, datum := range actualData {
		family := prom2json.Family{
			Name:    datum.Name,
			Help:    datum.Help,
			Type:    datum.Type,
			Metrics: make([]any, 0),
		}

		var metrics []any
		switch strings.ToLower(datum.Type) {
		case "histogram":
			metrics, err = convertHistogram(datum.Metrics)
		case "summary":
			metrics, err = convertSummary(datum.Metrics)
		default:
			metrics, err = convertMetric(datum.Metrics)
		}
		if err != nil {
			return nil, err
		}

		family.Metrics = append(family.Metrics, metrics...)
		expectedData = append(expectedData, &family)
	}

	return expectedData, nil
}

func convertMetric(actualMetrics []any) ([]any, error) {
	expected := make([]any, 0)
	for i := 0; i < len(actualMetrics); i++ {
		actual, ok := actualMetrics[i].(prom2json.Metric)
		if !ok {
			return nil, errors.New("actualMetric Type error")
		}
		actual.Value = GeZero
		actual.TimestampMs = GeZero
		expected = append(expected, actual)
	}
	return expected, nil
}

func convertHistogram(actualMetrics []any) ([]any, error) {
	expected := make([]any, 0)
	for i := 0; i < len(actualMetrics); i++ {
		actual, ok := actualMetrics[i].(prom2json.Histogram)
		if !ok {
			return nil, errors.New("actualMetric Type error")
		}
		actual.Sum = GeZero
		actual.TimestampMs = GeZero
		expected = append(expected, actual)
	}
	return expected, nil
}

func convertSummary(actualMetrics []any) ([]any, error) {
	expected := make([]any, 0)
	for i := 0; i < len(actualMetrics); i++ {
		actual, ok := actualMetrics[i].(prom2json.Summary)
		if !ok {
			return nil, errors.New("actualMetric Type error")
		}
		actual.Sum = GeZero
		actual.TimestampMs = GeZero
		expected = append(expected, actual)
	}
	return expected, nil
}
