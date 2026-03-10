package helpers

import (
	"bytes"
	"io"
	"net"
	"net/http"
)

func PostJSON(t TestReporter, endpoint string, body []byte, expectedStatus ...int) []byte {
	t.Helper()

	wantStatus := http.StatusOK
	if len(expectedStatus) > 0 {
		wantStatus = expectedStatus[0]
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s response: %v", endpoint, err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("unexpected status from %s: got=%d want=%d body=%s", endpoint, resp.StatusCode, wantStatus, string(respBody))
	}

	return respBody
}

func FreeTCPPort(t TestReporter) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp port: %v", err)
	}
	defer func() { _ = ln.Close() }()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected addr type: %T", ln.Addr())
	}

	return addr.Port
}
