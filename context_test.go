package httpc

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSetContext(t *testing.T) {
	ts := createGetServer(t)
	defer ts.Close()

	resp, err := dc().R().
		SetContext(context.Background()).
		Get(ts.URL + "/")

	assertError(t, err)
	assertEqual(t, http.StatusOK, resp.StatusCode())
	assertEqual(t, "200 OK", resp.Status())
	assertEqual(t, true, resp.Body() != nil)
	assertEqual(t, "TestGet: text response", resp.String())

	logResponse(t, resp)
}

func TestSetContextWithError(t *testing.T) {
	ts := createGetServer(t)
	defer ts.Close()

	resp, err := dcr().
		SetContext(context.Background()).
		Get(ts.URL + "/mypage")

	assertError(t, err)
	assertEqual(t, http.StatusBadRequest, resp.StatusCode())
	assertEqual(t, "", resp.String())

	logResponse(t, resp)
}

func TestSetContextCancel(t *testing.T) {
	ch := make(chan struct{})
	ts := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			ch <- struct{}{} // tell test request is finished
		}()
		t.Logf("Server: %v %v", r.Method, r.URL.Path)
		ch <- struct{}{}
		<-ch // wait for client to finish request
		n, err := w.Write([]byte("TestSetContextCancel: response"))
		// FIXME? test server doesn't handle request cancellation
		t.Logf("Server: wrote %d bytes", n)
		t.Logf("Server: err is %v ", err)
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-ch // wait for server to start request handling
		cancel()
	}()

	_, err := dc().R().
		SetContext(ctx).
		Get(ts.URL + "/")

	ch <- struct{}{} // tell server to continue request handling

	<-ch // wait for server to finish request handling

	t.Logf("Error: %v", err)
	if !errIsContextCanceled(err) {
		t.Errorf("Got unexpected error: %v", err)
	}
}

func TestSetContextCancelRetry(t *testing.T) {
	reqCount := 0
	ch := make(chan struct{})
	ts := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		defer func() {
			ch <- struct{}{} // tell test request is finished
		}()
		t.Logf("Server: %v %v", r.Method, r.URL.Path)
		ch <- struct{}{}
		<-ch // wait for client to finish request
		n, err := w.Write([]byte("TestSetContextCancel: response"))
		// FIXME? test server doesn't handle request cancellation
		t.Logf("Server: wrote %d bytes", n)
		t.Logf("Server: err is %v ", err)
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-ch // wait for server to start request handling
		cancel()
	}()

	c := dc().
		SetTimeout(time.Second * 3).
		SetRetryCount(3)

	_, err := c.R().
		SetContext(ctx).
		Get(ts.URL + "/")

	ch <- struct{}{} // tell server to continue request handling

	<-ch // wait for server to finish request handling

	t.Logf("Error: %v", err)
	if !errIsContextCanceled(err) {
		t.Errorf("Got unexpected error: %v", err)
	}

	if reqCount != 1 {
		t.Errorf("Request was retried %d times instead of 1", reqCount)
	}
}

func TestSetContextCancelWithError(t *testing.T) {
	ch := make(chan struct{})
	ts := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			ch <- struct{}{} // tell test request is finished
		}()
		t.Logf("Server: %v %v", r.Method, r.URL.Path)
		t.Log("Server: sending StatusBadRequest response")
		w.WriteHeader(http.StatusBadRequest)
		ch <- struct{}{}
		<-ch // wait for client to finish request
		n, err := w.Write([]byte("TestSetContextCancelWithError: response"))
		// FIXME? test server doesn't handle request cancellation
		t.Logf("Server: wrote %d bytes", n)
		t.Logf("Server: err is %v ", err)
	})
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-ch // wait for server to start request handling
		cancel()
	}()

	_, err := dc().R().
		SetContext(ctx).
		Get(ts.URL + "/")

	ch <- struct{}{} // tell server to continue request handling

	<-ch // wait for server to finish request handling

	t.Logf("Error: %v", err)
	if !errIsContextCanceled(err) {
		t.Errorf("Got unexpected error: %v", err)
	}
}

func TestClientRetryWithSetContext(t *testing.T) {
	var attemptctx int32
	ts := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Method: %v", r.Method)
		t.Logf("Path: %v", r.URL.Path)
		attp := atomic.AddInt32(&attemptctx, 1)
		if attp <= 4 {
			time.Sleep(time.Second * 2)
		}
		_, _ = w.Write([]byte("TestClientRetry page"))
	})
	defer ts.Close()

	c := dc().
		SetTimeout(time.Second * 1).
		SetRetryCount(3)

	_, err := c.R().
		SetContext(context.Background()).
		Get(ts.URL + "/")

	assertNotNil(t, ts)
	assertNotNil(t, err)
	assertEqual(t, true, (strings.HasPrefix(err.Error(), "Get "+ts.URL+"/") ||
		strings.HasPrefix(err.Error(), "Get \""+ts.URL+"/\"")))
}

func TestRequestContext(t *testing.T) {
	client := dc()
	r := client.NewRequest()
	assertNotNil(t, r.GetContext())

	r.SetContext(context.Background())
	assertNotNil(t, r.GetContext())
}

func errIsContextCanceled(err error) bool {
	ue, ok := err.(*url.Error)
	if !ok {
		return false
	}
	return ue.Err == context.Canceled
}
