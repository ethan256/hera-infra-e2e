package assert

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/apache/skywalking-infra-e2e/internal/components/assert"
	"github.com/apache/skywalking-infra-e2e/internal/config"
	"github.com/apache/skywalking-infra-e2e/internal/logger"
	"github.com/apache/skywalking-infra-e2e/internal/util"
	"github.com/apache/skywalking-infra-e2e/pkg/output"
)

var (
	assertQuery    string
	assertActual   string
	assertExpected string
	printer        output.Printer
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

type assertInfo struct {
	caseNumber int
	retryCount int
	interval   time.Duration
	failFast   bool
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

func concurrentlyAssertSingleCase(ctx context.Context, cancel context.CancelFunc, a *config.AssertCase, info *assertInfo) (res *output.CaseResult) {
	defer func() {
		if res.Err != nil && info.failFast {
			cancel()
		}
	}()

	if a.GetExpected() == "" {
		res.Msg = fmt.Sprintf("failed to assert %v:", caseName(a))
		res.Err = fmt.Errorf("the expected data file for %v is not specified", caseName(a))
		return res
	}

	for current := 0; current <= info.retryCount; current++ {
		select {
		case <-ctx.Done():
			res.Skip = true
			return res
		default:
			if err := assertSingleCase(a.GetExpected(), a.GetActual(), a.Query); err == nil {
				if current == 0 {
					res.Msg = fmt.Sprintf("asserted %v success\n", caseName(a))
				} else {
					res.Msg = fmt.Sprintf("asserted %v success, retried %d time(s)\n", caseName(a), current)
				}
				return res
			} else if current != info.retryCount {
				time.Sleep(info.interval)
			} else {
				res.Msg = fmt.Sprintf("failed to assert %v, retried %d time(s):", caseName(a), current)
				res.Err = err
			}
		}
	}

	return res
}

// assertCasesConcurrently assert the cases concurrently.
func assertCasesConcurrently(a *config.Assert, info *assertInfo) error {
	res := make([]*output.CaseResult, len(a.Cases))
	for i := range res {
		res[i] = &output.CaseResult{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for idx := range a.Cases {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Check if the context is canceled before asserting the case.
			select {
			case <-ctx.Done():
				res[i].Skip = true
				return
			default:
				// It's safe to do this, since each goroutine only modifies a single, different, designated slice element.
				res[i] = concurrentlyAssertSingleCase(ctx, cancel, &a.Cases[i], info)
			}
		}(idx)
	}
	wg.Wait()

	_, errNum, _ := printer.PrintResult(res)
	if errNum > 0 {
		return fmt.Errorf("failed to assert %d case(s)", errNum)
	}

	return nil
}

// assertCasesSerially verifies the cases serially.
func assertCasesSerially(a *config.Assert, info *assertInfo) (err error) {
	// A case may be skipped in fail-fast mode, so set it in advance.
	res := make([]*output.CaseResult, len(a.Cases))
	for i := range res {
		res[i] = &output.CaseResult{
			Skip: true,
		}
	}

	defer func() {
		_, errNum, _ := printer.PrintResult(res)
		if errNum > 0 {
			err = fmt.Errorf("failed to assert %d case(s)", errNum)
		}
	}()

	for idx := range a.Cases {
		printer.Start()
		v := &a.Cases[idx]

		if v.GetExpected() == "" {
			res[idx].Skip = false
			res[idx].Msg = fmt.Sprintf("failed to assert %v", caseName(v))
			res[idx].Err = fmt.Errorf("the expected data file for %v is not specified", caseName(v))

			printer.Warning(res[idx].Msg)
			printer.Fail(res[idx].Err.Error())
			if info.failFast {
				return
			}
			continue
		}

		for current := 0; current <= info.retryCount; current++ {
			if e := assertSingleCase(v.GetExpected(), v.GetActual(), v.Query); e == nil {
				if current == 0 {
					res[idx].Msg = fmt.Sprintf("assert %v \n", caseName(v))
				} else {
					res[idx].Msg = fmt.Sprintf("assert %v, retried %d time(s)\n", caseName(v), current)
				}
				res[idx].Skip = false
				printer.Success(res[idx].Msg)
				break
			} else if current != info.retryCount {
				if current == 0 {
					printer.UpdateText(fmt.Sprintf("failed to assert %v, will continue retry:", caseName(v)))
				} else {
					printer.UpdateText(fmt.Sprintf("failed to assert %v, retry [%d/%d]", caseName(v), current, info.retryCount))
				}
				time.Sleep(info.interval)
			} else {
				res[idx].Msg = fmt.Sprintf("failed to assert %v, retried %d time(s):", caseName(v), current)
				res[idx].Err = e
				res[idx].Skip = false
				printer.UpdateText(fmt.Sprintf("failed to assert %v, retry [%d/%d]", caseName(v), current, info.retryCount))
				printer.Warning(res[idx].Msg)
				printer.Fail(res[idx].Err.Error())
				if info.failFast {
					return
				}
			}
		}
	}

	return nil
}

func caseName(a *config.AssertCase) string {
	if a.Name == "" {
		if a.Actual != "" {
			return fmt.Sprintf("case[%s]", a.Actual)
		}
		return fmt.Sprintf("case[%s]", a.Query)
	}
	return a.Name
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
	failFast := e2eConfig.Assert.FailFast
	caseNumber := len(e2eConfig.Assert.Cases)

	info := assertInfo{
		caseNumber: caseNumber,
		retryCount: retryCount,
		interval:   interval,
		failFast:   failFast,
	}

	concurrency := e2eConfig.Assert.Concurrency
	if concurrency {
		// enable batch output mode when concurrency is enabled
		printer = output.NewPrinter(true)
		return assertCasesConcurrently(&e2eConfig.Assert, &info)
	}

	printer = output.NewPrinter(util.BatchMode)
	return assertCasesSerially(&e2eConfig.Assert, &info)
}

// TODO remove this in 2.0.0
func parseInterval(retryInterval any) (time.Duration, error) {
	var interval time.Duration
	var err error
	switch itv := retryInterval.(type) {
	case int:
		logger.Log.Warnf(`configuring assert.retry.interval with number is deprecated
and will be removed in future version, please use Duration style instead, such as 10s, 1m.`)
		interval = time.Duration(itv) * time.Millisecond
	case string:
		if interval, err = time.ParseDuration(itv); err != nil {
			return 0, err
		}
	case nil:
		interval = 0
	default:
		return 0, fmt.Errorf("failed to parse assert.retry.interval: %v", retryInterval)
	}
	if interval < 0 {
		interval = 1 * time.Second
	}
	return interval, nil
}
