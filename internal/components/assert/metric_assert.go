package assert

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/prometheus/prom2json"
)

// doMetricsAssert assert expected metrics and actual metrics.
func doMetricsAssert(expected, actual []*prom2json.Family) error {
	if len(expected) == 0 {
		return errors.New("expected metrics can not empty")
	}

	slices.SortFunc(expected, func(a, b *prom2json.Family) bool {
		return a.Name < b.Name
	})
	slices.SortFunc(actual, func(a, b *prom2json.Family) bool {
		return a.Name < b.Name
	})

	var err error
	// not contains metrics handle
	expected, err = handleNotContainersMetrics(expected, actual)
	if err != nil {
		return err
	}

	// 2. assert family [list]
	for expectedIndex, actualIndex := 0, 0; expectedIndex < len(expected); {
		// 无效匹配情形
		// 1, actualIndex 到达actual数组边界
		// 2, actual[actualIndex].Name > expected[expectedIndex].Name
		if actualIndex >= len(actual) || actual[actualIndex].Name > expected[expectedIndex].Name {
			return errors.Errorf("FamilyNotFoundError: expected[%s %s]", expected[expectedIndex].Name, expected[expectedIndex].Type)
		}
		// 指标类型或者指标名字不一致时，actualIndex应该往右移动
		if actual[actualIndex].Name != expected[expectedIndex].Name || actual[actualIndex].Type != expected[expectedIndex].Type {
			actualIndex++
			continue
		}
		// 匹配当前位置的expected和actual指标
		if err := assertFamilyMetric(expected[expectedIndex], actual[actualIndex]); err != nil {
			return err
		}
		// 当前指标匹配后，expectedIndex和actualIndex应该都往右移
		expectedIndex, actualIndex = expectedIndex+1, actualIndex+1
	}
	return nil
}

func handleNotContainersMetrics(expected, actual []*prom2json.Family) ([]*prom2json.Family, error) {
	expectedNotExistMetrics := make([]*prom2json.Family, 0)
	expectedMetrics := make([]*prom2json.Family, 0)

	for expectedIndex := 0; expectedIndex < len(expected); expectedIndex++ {
		splitName := strings.Split(expected[expectedIndex].Name, " ")
		if len(splitName) > 1 {
			var mark string
			mark, expected[expectedIndex].Name = splitName[0], splitName[1]
			if strings.EqualFold(mark, "notContains") {
				expectedNotExistMetrics = append(expectedNotExistMetrics, expected[expectedIndex])
			} else {
				expectedMetrics = append(expectedMetrics, expected[expectedIndex])
			}
		} else {
			expectedMetrics = append(expectedMetrics, expected[expectedIndex])
		}
	}

	if len(expectedNotExistMetrics) == 0 {
		return expectedMetrics, nil
	}

	errChan := make(chan error, len(expectedNotExistMetrics))
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var successAssertCount int

	for expectedIndex := 0; expectedIndex < len(expectedNotExistMetrics); expectedIndex++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			select {
			case <-stop:
				return
			default:
			}

			errChan <- doAssertNotContainsMetrics(expected[index], actual)
		}(expectedIndex)
	}

	for {
		select {
		case err := <-errChan:
			if err != nil {
				// 1:N模式，通知groutine退出
				close(stop)
				// 防止goroutine泄漏
				wg.Wait()
				return expectedMetrics, err
			}

			successAssertCount++
		default:
			if successAssertCount >= len(expectedNotExistMetrics) {
				return expectedMetrics, nil
			}
		}
	}
}

func doAssertNotContainsMetrics(expectedFamily *prom2json.Family, actual []*prom2json.Family) error {
	re := regexp.MustCompile(expectedFamily.Name)

	for actualIndex := 0; actualIndex < len(actual); actualIndex++ {
		// 校验指标name
		if re.MatchString(actual[actualIndex].Name) {
			return errors.Errorf("expected notContains Metrics[`%q`]", expectedFamily.Name)
		}
	}
	return nil
}

// assertFamilyMetric metrics separately according to metric type.
func assertFamilyMetric(expectedFamily, actualFamily *prom2json.Family) error {
	var err error
	switch strings.ToLower(expectedFamily.Type) {
	case "summary":
		err = doAssertFunc(expectedFamily.Metrics, actualFamily.Metrics, assertSummary)
	case "histogram":
		err = doAssertFunc(expectedFamily.Metrics, actualFamily.Metrics, assertHistogram)
	default:
		err = doAssertFunc(expectedFamily.Metrics, actualFamily.Metrics, assertMetric)
	}
	return errors.Wrap(err, fmt.Sprintf("Metric[%s]", expectedFamily.Name))
}

func doAssertFunc(expectedMetrics, actualMetrics []any,
	fn func(expectedIndex int, expectMetric map[string]any, actualMetrics []any, matched *sync.Map) error) error {
	errChan := make(chan error, len(expectedMetrics))
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var matched sync.Map
	var successAssertCount int

	for expectedIndex := 0; expectedIndex < len(expectedMetrics); expectedIndex++ {
		wg.Add(1)
		go func(expectedIndex int) {
			defer wg.Done()
			// 通过close stop通道来提前退出当前goroutine
			select {
			case <-stop:
				return
			default:
			}

			expectedMetricMap, ok := expectedMetrics[expectedIndex].(map[string]any)
			if !ok {
				errChan <- errors.New("expectedMetric type error, want a map[string]any")
				return
			}
			errChan <- fn(expectedIndex, expectedMetricMap, actualMetrics, &matched)
		}(expectedIndex)
	}

	for {
		select {
		case err := <-errChan:
			if err != nil {
				close(stop)
				// 防止goroutine泄漏
				wg.Wait()
				// 只返回通道里第一个非nil的错误
				return err
			}
			successAssertCount++
		default:
			if successAssertCount >= len(expectedMetrics) {
				return nil
			}
		}
	}
}

func assertMetric(expectedIndex int, expectMetric map[string]any, actualMetrics []any, matched *sync.Map) error {
	var expected prom2json.Metric
	var err error
	if err = mapstructure.Decode(&expectMetric, &expected); err != nil {
		return errors.Wrapf(err, "expectedMetric type error")
	}

	// maxIndex 表示actualIndex的最大值，
	// actualIndex的初始值为expectedIndex，目的是优先匹配相同位置的metrics，因为大多数情况下expected和actual顺序是是相同的
	maxIndex := len(actualMetrics) + expectedIndex

	for actualIndex := expectedIndex; actualIndex < maxIndex; actualIndex++ {
		if actualIndex >= len(actualMetrics) {
			actualIndex -= len(actualMetrics)
		}

		actual, ok := actualMetrics[actualIndex].(prom2json.Metric)
		if !ok {
			return errors.New("actualMetric Type error")
		}

		if _, ok := matched.Load(actualIndex); ok {
			continue
		}
		if err = assertHelper(expected.Value, actual.Value, "", "", "", "", expected.Labels, actual.Labels); err == nil {
			matched.Store(actualIndex, actualIndex)
			break
		}
	}
	return err
}

func assertSummary(expectedIndex int, expectMetric map[string]any, actualMetrics []any, matched *sync.Map) error {
	var expected prom2json.Summary
	var err error
	if err = mapstructure.Decode(&expectMetric, &expected); err != nil {
		return errors.Wrapf(err, "expectedMetric type error")
	}

	// maxIndex 表示actualIndex的最大值，
	// actualIndex的初始值为expectedIndex，目的是优先匹配相同位置的metrics，因为大多数情况下expected和actual顺序是是相同的
	maxIndex := len(actualMetrics) + expectedIndex

	for actualIndex := expectedIndex; actualIndex < maxIndex; actualIndex++ {
		if actualIndex >= len(actualMetrics) {
			actualIndex -= len(actualMetrics)
		}

		actual, ok := actualMetrics[actualIndex].(prom2json.Summary)
		if !ok {
			return errors.New("actualMetric Type error")
		}

		if _, ok := matched.Load(actualIndex); ok {
			continue
		}
		if err = assertHelper("", "", expected.Count, actual.Count, expected.Sum, actual.Sum, expected.Labels, actual.Labels); err == nil {
			matched.Store(actualIndex, actualIndex)
			break
		}
	}
	return err
}

func assertHistogram(expectedIndex int, expectMetric map[string]any, actualMetrics []any, matched *sync.Map) error {
	var expected prom2json.Histogram
	var err error
	if err = mapstructure.Decode(&expectMetric, &expected); err != nil {
		return errors.Wrapf(err, "expectedMetric type error")
	}

	// maxIndex 表示actualIndex的最大值，
	// actualIndex的初始值为expectedIndex，目的是优先匹配相同位置的metrics，因为大多数情况下expected和actual顺序是是相同的
	maxIndex := len(actualMetrics) + expectedIndex

	for actualIndex := expectedIndex; actualIndex < maxIndex; actualIndex++ {
		if actualIndex >= len(actualMetrics) {
			actualIndex -= len(actualMetrics)
		}

		actual, ok := actualMetrics[actualIndex].(prom2json.Histogram)
		if !ok {
			return errors.New("actualMetric Type error")
		}

		if _, ok := matched.Load(actualIndex); ok {
			continue
		}
		if err = assertHelper("", "", expected.Count, actual.Count, expected.Sum, actual.Sum, expected.Labels, actual.Labels); err == nil {
			matched.Store(actualIndex, actualIndex)
			break
		}
	}
	return err
}

func assertHelper(expectedValue, actualValue, expectedCount, actualCount, expectedSum, actualSum string,
	expectedLabels, actualLabels map[string]string) (err error) {
	if len(actualLabels) < len(expectedLabels) {
		return fmt.Errorf("len(actualLabels) < len(expectedLabels), expected: %+v, actual: %+v", expectedLabels, actualLabels)
	}
	if err = assertLabels(expectedLabels, actualLabels); err != nil {
		return
	}

	if err = ValueAssert("Value", expectedValue, actualValue); err != nil {
		return
	}

	if err = ValueAssert("Count", expectedCount, actualCount); err != nil {
		return
	}

	if err = ValueAssert("Sum", expectedSum, actualSum); err != nil {
		return
	}

	return nil
}

func assertLabels(expectedLabels, actualLabels map[string]string) error {
	var err error
	for k, v := range expectedLabels {
		if err = ValueAssert(k, v, actualLabels[k]); err != nil {
			err = errors.Wrapf(err, "Labels Diff")
			return err
		}
	}
	return nil
}
