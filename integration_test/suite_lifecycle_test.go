package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/integration_test/helpers"
	. "github.com/onsi/ginkgo/v2"
)

var (
	suiteMongoHost  string
	suiteRabbitHost string
	suiteRabbitPort int
	suiteClockHost  string
	suiteClockPort  int
	suiteAppPort    int
	suiteStopMain   func()
	suiteRunErrCh   <-chan error
	suiteReady      bool
)

var _ = BeforeSuite(func() {
	t := GinkgoT()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	DeferCleanup(cancel)

	suiteMongoHost, suiteRabbitHost, suiteRabbitPort = helpers.StartContainers(ctx)
	suiteClockHost, suiteClockPort = helpers.StartClockEmulator(t)

	suiteAppPort = helpers.FreeTCPPort(t)
	suiteStopMain, suiteRunErrCh = helpers.ApplicationStart(
		t,
		suiteAppPort,
		suiteClockHost,
		suiteClockPort,
		suiteMongoHost,
		suiteRabbitHost,
		suiteRabbitPort,
	)

	if err := waitForHTTP200OrExit(suiteAppPort, suiteRunErrCh, 30*time.Second); err != nil {
		Skip(fmt.Sprintf("skipping integration test: %v", err))
	}
	suiteReady = true
})

var _ = AfterSuite(func() {
	if !suiteReady || suiteStopMain == nil {
		return
	}

	suiteStopMain()
	select {
	case err := <-suiteRunErrCh:
		if err != nil {
			Fail(fmt.Sprintf("main returned error: %v", err))
		}
	case <-time.After(45 * time.Second):
		Fail("main did not stop after context cancel")
	}
})

func waitForHTTP200OrExit(appPort int, runErrCh <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", appPort)
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case runErr := <-runErrCh:
			if runErr != nil {
				return fmt.Errorf("main exited before ready: %w", runErr)
			}
			return fmt.Errorf("main exited before ready")
		default:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s to return 200", url)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
