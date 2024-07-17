package httpc

import (
	"context"
	"crypto/tls"
	"net"
	"net/http/httptrace"
	"time"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// TraceInfo struct
//_______________________________________________________________________

// TraceInfo struct is used provide request trace info such as DNS lookup
// duration, Connection obtain duration, Server processing duration, etc.
//
// Since v2.0.0
type TraceInfo struct {
	// DNSLookup DNS解析的时间
	DNSLookup time.Duration

	// ConnTime is a duration that took to obtain a successful connection.
	ConnTime time.Duration

	// TCPConnTime TCP连接持续的时间
	TCPConnTime time.Duration

	// TLSHandshake TLS握手持续的时间
	TLSHandshake time.Duration

	// ServerTime is a duration that server took to respond first byte.
	ServerTime time.Duration

	// ResponseTime is a duration since first response byte from server to
	// request completion.
	ResponseTime time.Duration

	// TotalTime 请求总计花费时间
	TotalTime time.Duration

	// IsConnReused is whether this connection has been previously
	// used for another HTTP request.
	IsConnReused bool

	// IsConnWasIdle is whether this connection was obtained from an
	// idle pool.
	IsConnWasIdle bool

	// ConnIdleTime is a duration how long the connection was previously
	// idle, if IsConnWasIdle is true.
	ConnIdleTime time.Duration

	// RequestAttempt is to represent the request attempt made during a Resty
	// request execution flow, including retry count.
	RequestAttempt int

	// RemoteAddr returns the remote network address.
	RemoteAddr net.Addr
}

type ConnectInfo struct {
	//准备建立连接
	GetConnectTime time.Time `json:"getConnectTime"`
	//成功建立连接的时间
	GotConnectTime time.Time `json:"gotConnectTime"`

	//接收到响应报文的时间
	ReceiveHttpResponseTime time.Time `json:"receiveHttpResponseTime"`

	//连接断开的时间
	ConnectDone time.Time `json:"connectDone"`

	//请求目标的地址
	RemoteAddr net.Addr `json:"remoteAddr"`

	//请求目标的源地址
	LocalAddr net.Addr `json:"localAddr"`
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ClientTrace struct and its methods
//_______________________________________________________________________

// tracer struct maps the `httptrace.ClientTrace` hooks into Fields
// with same naming for easy understanding. Plus additional insights
// Request.
type clientTrace struct {
	getConn              time.Time
	dnsStart             time.Time
	dnsDone              time.Time
	connectDone          time.Time
	tlsHandshakeStart    time.Time
	tlsHandshakeDone     time.Time
	gotConn              time.Time
	gotFirstResponseByte time.Time
	endTime              time.Time
	gotConnInfo          httptrace.GotConnInfo
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Trace unexported methods
//_______________________________________________________________________

func (t *clientTrace) createContext(ctx context.Context) context.Context {
	return httptrace.WithClientTrace(
		ctx,
		&httptrace.ClientTrace{
			DNSStart: func(_ httptrace.DNSStartInfo) {
				t.dnsStart = time.Now()
			},
			DNSDone: func(_ httptrace.DNSDoneInfo) {
				t.dnsDone = time.Now()
			},
			ConnectStart: func(_, _ string) {
				if t.dnsDone.IsZero() {
					t.dnsDone = time.Now()
				}
				if t.dnsStart.IsZero() {
					t.dnsStart = t.dnsDone
				}
			},
			ConnectDone: func(net, addr string, err error) {
				t.connectDone = time.Now()
			},
			//准备获取tcp连接的时候调用
			GetConn: func(_ string) {
				t.getConn = time.Now()
			},
			//获取到tcp连接的时候调用
			GotConn: func(ci httptrace.GotConnInfo) {
				t.gotConn = time.Now()
				t.gotConnInfo = ci
			},
			GotFirstResponseByte: func() {
				t.gotFirstResponseByte = time.Now()
			},
			TLSHandshakeStart: func() {
				t.tlsHandshakeStart = time.Now()
			},
			TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
				t.tlsHandshakeDone = time.Now()
			},
		},
	)
}
