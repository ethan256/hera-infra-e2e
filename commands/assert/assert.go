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
		if assertExpected != "" {
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
	if err = assert.DataAssert(expectedFile, actualFile, query); err != nil {
		return errors.Errorf("Assert Failed: expectedFile: %s, actualFile: %s, exception: %v", expectedFile, actualFile, err)
	}
	logger.Log.Infof("assert the actualFile: %s", actualFile)
	return
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
	interval, err := time.ParseDuration(e2eConfig.Assert.RetryStrategy.Interval)
	if err != nil {
		return err
	}

	for idx, v := range e2eConfig.Assert.Cases {
		if v.GetExpected() == "" {
			return fmt.Errorf("the expected data file for case[%v] is not specified", idx)
		}
		for current := 1; current <= retryCount; current++ {
			if err := assertSingleCase(v.GetExpected(), v.GetActual(), v.Query); err == nil {
				break
			} else if current != retryCount {
				logger.Log.Warnf("assert case failure, will continue retry, %v", err)
				time.Sleep(interval)
			} else {
				return err
			}
		}
	}

	return nil
}
