package assert

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/prometheus/prom2json"
)

// MetricsAssert assert expected metrics and actual metrics.
func MetricsAssert(expected, actual []*prom2json.Family) error {
	if len(expected) == 0 {
		return nil
	}

	// 1. assert family length
	if len(expected) > len(actual) {
		return errors.Errorf("actual metrics length must ge expected metrics length: expexted=>%d, actual=>%d", len(expected), len(actual))
	}

	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Name > expected[j].Name
	})
	sort.Slice(actual, func(i, j int) bool {
		return actual[i].Name > actual[j].Name
	})
	// 2. assert family [list]
	for i, j := 0, 0; i < len(expected); {
		if j >= len(actual) {
			return errors.Errorf("FamilyNotFoundError: expected[%s %s]", expected[i].Name, expected[i].Type)
		}
		if actual[j].Name != expected[i].Name || actual[j].Type != expected[i].Type {
			j++
			continue
		}
		if err := assertFamilyMetric(expected[i], actual[j]); err != nil {
			return err
		}
		i, j = i+1, j+1
	}
	return nil
}

// assertFamilyMetric metrics separately according to metric type.
func assertFamilyMetric(expectedFamily, actualFamily *prom2json.Family) error {
	var err error
	switch strings.ToLower(expectedFamily.Type) {
	case "summary":
		err = summaryAssert(expectedFamily.Metrics, actualFamily.Metrics)
	case "histogram":
		err = histogramAssert(expectedFamily.Metrics, actualFamily.Metrics)
	default:
		err = metricAssert(expectedFamily.Metrics, actualFamily.Metrics)
	}
	return errors.Wrap(err, fmt.Sprintf("Metric[%s]", expectedFamily.Name))
}

func metricAssert(expectedMetrics, actualMetrics []interface{}) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(expectedMetrics))
	for i := 0; i < len(expectedMetrics); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m, ok := expectedMetrics[i].(map[string]interface{})
			if !ok {
				errChan <- errors.New("expectedMetric type error, want a map[string]interface{}")
				return
			}
			var expected prom2json.Metric
			if err := mapstructure.Decode(&m, &expected); err != nil {
				errChan <- errors.Wrapf(err, "expectedMetric type error")
				return
			}
			actual, ok := actualMetrics[i].(prom2json.Metric)
			if !ok {
				errChan <- errors.New("actualMetric Type error")
				return
			}

			if err := ValueAssert("Value", expected.Value, actual.Value); err != nil {
				errChan <- err
				return
			}

			for k, v := range expected.Labels {
				if err := ValueAssert(k, v, actual.Labels[k]); err != nil {
					errChan <- errors.Wrapf(err, "Labels Diff")
					return
				}
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

func summaryAssert(expectedMetrics, actualMetrics []interface{}) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(expectedMetrics))
	for i := 0; i < len(expectedMetrics); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m, ok := expectedMetrics[i].(map[string]interface{})
			if !ok {
				errChan <- errors.New("expectedMetric type error, want a map[string]interface{}")
				return
			}
			var expected prom2json.Summary
			if err := mapstructure.Decode(&m, &expected); err != nil {
				errChan <- errors.Wrapf(err, "expectedMetric type error")
				return
			}
			actual, ok := actualMetrics[i].(prom2json.Summary)
			if !ok {
				errChan <- errors.New("actualMetric Type error")
				return
			}

			if err := assertHelper(expected.Count, actual.Count, expected.Sum, actual.Sum, expected.Labels, actual.Labels,
				expected.Quantiles, actual.Quantiles, nil, nil); err != nil {
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

func histogramAssert(expectedMetrics, actualMetrics []interface{}) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(expectedMetrics))
	for i := 0; i < len(expectedMetrics); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m, ok := expectedMetrics[i].(map[string]interface{})
			if !ok {
				errChan <- errors.New("expectedMetric type error, want a map[string]interface{}")
				return
			}
			var expected prom2json.Histogram
			if err := mapstructure.Decode(&m, &expected); err != nil {
				errChan <- errors.Wrapf(err, "expectedMetric type error")
				return
			}
			actual, ok := actualMetrics[i].(prom2json.Histogram)
			if !ok {
				errChan <- errors.New("actualMetric Type error")
				return
			}

			if err := assertHelper(expected.Count, actual.Count, expected.Sum, actual.Sum, expected.Labels, actual.Labels,
				nil, nil, expected.Buckets, actual.Buckets); err != nil {
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

func assertHelper(expectedCount, actualCount, expectedSum, actualSum string, expectedLabels, actualLabels map[string]string,
	expectedQuantiles, actualQuantiles map[string]string, expectedBuckets, actualBuckets map[string]string) (err error) {
	if err = ValueAssert("Count", expectedCount, actualCount); err != nil {
		return
	}
	if err = ValueAssert("Sum", expectedSum, actualSum); err != nil {
		return
	}

	for k, v := range expectedLabels {
		if err = ValueAssert(k, v, actualLabels[k]); err != nil {
			err = errors.Wrapf(err, "Labels Diff")
			return
		}
	}

	if len(expectedQuantiles) != len(actualQuantiles) {
		err = errors.Errorf("Quantiles Diff: expected length=>%d, actual length=>%d", len(expectedQuantiles), len(actualQuantiles))
		return
	}

	if len(expectedBuckets) != len(actualBuckets) {
		err = errors.Errorf("Buckets Diff: expected length=>%d, actual length=>%d", len(expectedBuckets), len(actualBuckets))
		return
	}
	return nil
}
