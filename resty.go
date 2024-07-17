package httpc

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/cookiejar"
	"runtime"
	"time"

	"golang.org/x/net/publicsuffix"
)

// Version # of resty
const Version = "2.7.0"

// New method creates a new Resty client.
func New() *Client {
	cookieJar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return createClient(&http.Client{
		Jar: cookieJar,
	})
}

// NewWithClient method creates a new Resty client with given `http.Client`.
func NewWithClient(hc *http.Client) *Client {
	return createClient(hc)
}

// NewWithLocalAddr method creates a new Resty client with given Local Address
// to dial from.
func NewWithLocalAddr(localAddr net.Addr) *Client {
	cookieJar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return createClient(&http.Client{
		Jar:       cookieJar,
		Transport: createTransport(localAddr),
	})
}

// NewDefaultClient 兼容shttp的默认http客户端
func NewDefaultClient() *Client {
	return NewDefaultRedirectClient().DisableRedirect()
}

// NewDefaultRedirectClient 兼容shttp的跳转客户端
func NewDefaultRedirectClient() *Client {
	cookieJar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	dialer := &net.Dialer{
		Timeout:   3 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	tr := &http.Transport{
		//Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          50,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives:     false,
	}
	clietn := createClient(&http.Client{
		Jar:       cookieJar,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	})

	return clietn
}
