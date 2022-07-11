package assert

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

var (
	assertQuery    string
	assertActual   string
	assertExpected string
)

func init() {
	Assert.Flags().StringVarP(&assertQuery, "assert-query", "", "", "the assert-query to get the actual data")
	Assert.Flags().StringVarP(&assertActual, "assert-actual", "", "", "the assert-actual data file, only JSON file format is supported")
	Assert.Flags().StringVarP(&assertExpected, "assert-expected", "", "", "the assert-expected data file, only JSON file format is supported")
}

var Assert = &cobra.Command{
	Use:   "assert",
	Short: "assert if the actual data match the expected data",
	RunE: func(cmd *cobra.Command, args []string) error {
		if assertExpected != "" || assertQuery != "" {
			return assertSingleCase(assertExpected, assertActual, assertQuery)
		}
		// If there is no given flags.
		return DoAssertAccordingConfig()
	},
}

func assertSingleCase(expectedFile, actualFile, query string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error("`assertSingleCase` func throws a panic, we are recover")
			// check exactly what the panic was and create error.
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknow panic")
			}
		}
	}()

	if query != "" {
		err = assert.MetricsAssert(expectedFile, query)
		if err != nil {
			return errors.Wrap(err, "assert metrics failed")
		}
	} else {
		err = assert.TracesAssert(expectedFile, actualFile)
		if err != nil {
			return errors.Wrap(err, "assert traces failed")
		}
	}
	return nil
}

// DoAssertAccordingConfig reads cases from the config file and assert them.
func DoAssertAccordingConfig() error {
	if config.GlobalConfig.Error != nil {
		return config.GlobalConfig.Error
	}

	e2eConfig := config.GlobalConfig.E2EConfig

	retryCount := e2eConfig.Assert.RetryStrategy.Count
	if retryCount <= 0 {
		retryCount = 1
	}
	interval, err := parseInterval(e2eConfig.Assert.RetryStrategy.Interval)
	if err != nil {
		return err
	}

	for idx, v := range e2eConfig.Assert.Cases {
		if v.GetExpected() == "" {
			return fmt.Errorf("the expected data file for case[%v] is not specified", idx)
		}
		for current := 0; current <= retryCount; current++ {
			if err := assertSingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
				break
			} else if current != retryCount {
				logger.Log.Warnf("assert case[%d] failure, will continue after %ds", idx, interval/time.Second)
				time.Sleep(interval)
			} else {
				logger.Log.Errorf("assert case[%d] failure, will exit, error: %v", idx, err)
				return err
			}
		}
	}

	return nil
}

// TODO remove this in 2.0.0
func parseInterval(retryInterval any) (time.Duration, error) {
	var interval time.Duration
	var err error
	switch itv := retryInterval.(type) {
	case int:
		logger.Log.Warnf(`configuring verify.retry.interval with number is deprecated
and will be removed in future version, please use Duration style instead, such as 10s, 1m.`)
		interval = time.Duration(itv) * time.Millisecond
	case string:
		if interval, err = time.ParseDuration(itv); err != nil {
			return 0, err
		}
	case nil:
		interval = 0
	default:
		return 0, fmt.Errorf("failed to parse verify.retry.interval: %v", retryInterval)
	}
	if interval < 0 {
		interval = 1 * time.Second
	}
	return interval, nil
}
