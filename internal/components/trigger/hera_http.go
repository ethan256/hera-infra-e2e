package trigger

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/apache/skywalking-infra-e2e/internal/logger"
)

type heraHTTPAction struct {
	interval      time.Duration
	times         int
	url           string
	method        string
	body          string
	headers       map[string]string
	executedCount int
	stopCh        chan struct{}
	client        *http.Client
}

func NewHeraHTTPAction(intervalStr string, times int, url, method, body string, headers map[string]string) (Action, error) {
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, err
	}

	if interval <= 0 {
		return nil, fmt.Errorf("trigger interval should be > 0, but was %s", interval)
	}

	// there can be env variables in url, say, "http://${GATEWAY_HOST}:${GATEWAY_PORT}/test"
	url = os.ExpandEnv(url)

	return &heraHTTPAction{
		interval:      interval,
		times:         times,
		url:           url,
		method:        strings.ToUpper(method),
		body:          body,
		headers:       headers,
		executedCount: 0,
		stopCh:        make(chan struct{}, 1),
		client:        &http.Client{},
	}, nil
}

func (h *heraHTTPAction) Do() chan error {
	t := time.NewTicker(h.interval)

	logger.Log.Infof("trigger will request URL %s %d times with interval %s.", h.url, h.times, h.interval)

	result := make(chan error)
	go func() {
		for {
			select {
			case <-t.C:
				err := h.execute()
				// send nil to result channel, then stop ticker, when request success.
				// Otherwise, retry send http request, until send err to result when `h.times == h.executedCount`.
				if err == nil || h.times == h.executedCount {
					result <- err
					t.Stop()
					return
				}
			case <-h.stopCh:
				t.Stop()
				result <- nil
				return
			}
		}
	}()

	return result
}

func (h *heraHTTPAction) Stop() {
	h.stopCh <- struct{}{}
}

func (h *heraHTTPAction) request() (*http.Request, error) {
	request, err := http.NewRequest(h.method, h.url, strings.NewReader(h.body))
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	for k, v := range h.headers {
		headers[k] = []string{v}
	}
	request.Header = headers
	return request, err
}

func (h *heraHTTPAction) execute() error {
	req, err := h.request()
	if err != nil {
		logger.Log.Errorf("failed to create new request %v", err)
		return err
	}
	logger.Log.Debugf("request URL %s the %d time.", h.url, h.executedCount)
	response, err := h.client.Do(req)
	h.executedCount++
	if err != nil {
		logger.Log.Errorf("do request exception %v", err)
		return err
	}
	_, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()

	logger.Log.Debugf("do request %v response http code %v", h.url, response.StatusCode)
	if response.StatusCode == http.StatusOK {
		logger.Log.Debugf("do http action %+v success.", *h)
		return nil
	}
	return fmt.Errorf("do request failed, response status code: %d", response.StatusCode)
}
