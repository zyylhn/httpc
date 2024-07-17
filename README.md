该项目改自https://github.com/go-resty/resty

与公司内部shttp兼容request和response基本兼容，client的使用方式并不兼容，但是会尽可能提供出来相关构造函数创建出和shttp功能相同的client

### Client

#### 新建一个httpc.Client

```go
httpc.New() *Client{......}     //返回一个默认的httpc.Client
httpc.NewWithClient(hc *http.Client) *Client   //根据http.Client生成httpc.Client
httpc.NewWithLocalAddr(localAddr net.Addr) *Client  //使用指定地址生成一个clien
```

#### Middleware/Hook/Callback

```go
// RequestMiddleware 中间件，请求发送之前调用
	RequestMiddleware func(*Client, *Request) error

	// ResponseMiddleware 中间件，接收到响应调用
	ResponseMiddleware func(*Client, *Response) error

	// PreRequestHook request的hook, 在准备发送请求的时候调用
	PreRequestHook func(*Client, *http.Request) error

	// RequestLogCallback 用于请求日志，在请求记录之前调用
	RequestLogCallback func(*RequestLog) error

	// ResponseLogCallback 用于响应日志，在记录响应之前调用
	ResponseLogCallback func(*ResponseLog) error

	// ErrorHook 用于对请求错误做出反应，在尝试所有重试后调用
	ErrorHook func(*Request, error)

	// SuccessHook 用于对请求成功作出反应
	SuccessHook func(*Client, *Response)
```

client设置,可以通过相关函数设置以下属性

```go
  beforeRequest       []RequestMiddleware
	udBeforeRequest     []RequestMiddleware
	preReqHook          PreRequestHook
	successHooks        []SuccessHook
	afterResponse       []ResponseMiddleware
	requestLog          RequestLogCallback
	responseLog         ResponseLogCallback
	errorHooks          []ErrorHook
	invalidHooks        []ErrorHook
	panicHooks          []ErrorHook
```

默认新建的client存在部分中间件来满足请求的基本功能（//todo 具体含义待查看完善）

```go
// default before request middlewares
	c.beforeRequest = []RequestMiddleware{
		parseRequestURL,
		parseRequestHeader,
		parseRequestBody,
		createHTTPRequest,     //利用设置的信息创建http.requeset
		addCredentials,
	}

	// user defined request middlewares
	c.udBeforeRequest = []RequestMiddleware{}

	// default after response middlewares
	c.afterResponse = []ResponseMiddleware{
		responseLogger,
		parseResponseBody,
		saveResponseIntoFile,
	}
```

#### Transport

默认设置已经设置好了相关请求Transport。

```go
func createTransport(localAddr net.Addr) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	if localAddr != nil {
		dialer.LocalAddr = localAddr
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
}
```

也可以使用client.SetTransport自行设置

```go
func (c *Client) SetTransport(transport http.RoundTripper) *Client {
	if transport != nil {
		c.httpClient.Transport = transport
	}
	return c
}
```

#### 代理

```go
func (c *Client) SetProxy(proxyURL string) *Client {
	/*........*/
	return c
}

func (c *Client) RemoveProxy() *Client {
	/*........*/
	return c
}

func (c *Client) IsProxySet() bool {
	return c.proxyURL != nil
}
```

#### TLS配置

```go
func (c *Client) SetTLSClientConfig(config *tls.Config) *Client {
	transport, err := c.Transport()
	if err != nil {
		c.log.Errorf("%v", err)
		return c
	}
	transport.TLSClientConfig = config
	return c
}
```



#### 其他

包含很多可以设置request的函数，用于设置每个请求的参数。当然在request中同样设置了的话会将其覆盖。仔细看过后会发现基本上request能设置的参数client都可以全局设置，在每一个请求中生效

### Request

新建一个httpc.request

```go
func (c *Client) NewRequest() *Request {
	return c.R()
}

func (c *Client) R() *Request {
	/*......*/
}
```

新建之后的httpc.request中保存了client，所以不需在使用client去发送请求，直接使用封装好的方法去发送请求

```go
// Get method does GET HTTP request. It's defined in section 4.3.1 of RFC7231.
func (r *Request) Get(url string) (*Response, error) {
	return r.Execute(MethodGet, url)
}

// Head method does HEAD HTTP request. It's defined in section 4.3.2 of RFC7231.
func (r *Request) Head(url string) (*Response, error) {
	return r.Execute(MethodHead, url)
}

// Post method does POST HTTP request. It's defined in section 4.3.3 of RFC7231.
func (r *Request) Post(url string) (*Response, error) {
	return r.Execute(MethodPost, url)
}

// Put method does PUT HTTP request. It's defined in section 4.3.4 of RFC7231.
func (r *Request) Put(url string) (*Response, error) {
	return r.Execute(MethodPut, url)
}

// Delete method does DELETE HTTP request. It's defined in section 4.3.5 of RFC7231.
func (r *Request) Delete(url string) (*Response, error) {
	return r.Execute(MethodDelete, url)
}

// Options method does OPTIONS HTTP request. It's defined in section 4.3.7 of RFC7231.
func (r *Request) Options(url string) (*Response, error) {
	return r.Execute(MethodOptions, url)
}

// Patch method does PATCH HTTP request. It's defined in section 2 of RFC5789.
func (r *Request) Patch(url string) (*Response, error) {
	return r.Execute(MethodPatch, url)
}
```

#### 设置请求内容

简单的就不赘述了（设置请求参数、路径、header.....）

##### 设置body

```go
func (r *Request) SetBody(body interface{}) *Request {
	/*........*/
	return r
}
```

支持以string、[]byte、map[string]interface{}、struct、io.Reader格式输入

当是struct、map、slice的时候将会根据content type给序列化（json和xml，xml仅序列化struct），匹配规则如下（json:application/json）

```go
jsonCheck = regexp.MustCompile(`(?i:(application|text)/(json|.*\+json|json\-.*)(;|$))`)
	xmlCheck  = regexp.MustCompile(`(?i:(application|text)/(xml|.*\+xml)(;|$))`)
```

其他格式会自动转换成body

##### 设置http验证相关内容

常见http身份验证请求格式

```
Authorization: Basic <base64-encoded-username-and-password>
Authorization: Basic dXNlcjpwYXNz       //其中，"dXNlcjpwYXNz" 是 "user:pass" 的 Base64 编码结果

Authorization: Bearer <access-token>
Authorization: Bearer abcdefg12345      "abcdefg12345" 是访问令牌。

Authorization: NTLM <type-1-message>
Authorization: NTLM TlRMTVNTUAABAAAAB4IIAAAAAAAAAAAAAAAAAAAAAAA=    中，"TlRMTVNTUAABAAAAB4IIAAAAAAAAAAAAAAAAAAAAAAA=" 是 Base64 编码的 Type 1 消息

Authorization: Digest username="<username>", realm="<realm>", nonce="<nonce>", uri="<uri>", response="<response>", opaque="<opaque>", qop="<qop>", nc="<nc>", cnonce="<cnonce>"
Authorization: Digest username="user", realm="example.com", nonce="5fcbd5b5a7b2c", uri="/", response="5d5c3f3d82f5b5d5", opaque="", qop="auth", nc="00000001", cnonce="2c187d788be27b4c"
其中，各个参数的含义如下：
username：用户名
realm：认证领域
nonce：服务器生成的随机数
uri：请求的资源路径
response：由客户端计算出的响应字符串
opaque：不透明字符串，由服务器生成
qop：质量保证（Quality of Protection），可以是 "auth" 或 "auth-int"
nc：请求计数器，防止重放攻击
cnonce：客户端生成的随机数
```

```go
func (r *Request) SetBasicAuth(username, password string) *Request {
   r.UserInfo = &User{Username: username, Password: password}
   return r
}

func (r *Request) SetAuthToken(token string) *Request {
	r.Token = token
	return r
}

func (r *Request) SetAuthScheme(scheme string) *Request {
	r.AuthScheme = scheme
	return r
}

func (r *Request) SetDigestAuth(username, password string) *Request {
	oldTransport := r.client.httpClient.Transport
	r.client.OnBeforeRequest(func(c *Client, _ *Request) error {
		c.httpClient.Transport = &digestTransport{
			digestCredentials: digestCredentials{username, password},
			transport:         oldTransport,
		}
		return nil
	})
	r.client.OnAfterResponse(func(c *Client, _ *Response) error {
		c.httpClient.Transport = oldTransport
		return nil
	})

	return r
}
```

对比可以发现，Basic和Digest需要用户名密码，可以直接通过函数设置

另外两种可以设置协议和Tocken实现

##### 这只ctx

```go
func (r *Request) SetContext(ctx context.Context) *Request {
	/*.......*/
	return r
}
```



#### 其他功能

##### 自动解析响应body

```go
//可以通过在此设置成功响应的数据结构
func (r *Request) SetResult(res interface{}) *Request {
	if res != nil {
		r.Result = getPointer(res)
	}
	return r
}

//可在此设置失败响应的数据结构
func (r *Request) SetError(err interface{}) *Request {
	r.Error = getPointer(err)
	return r
}
```

在请求响应中的中间件（afterResponse  []ResponseMiddleware）中的parseResponseBody()中解析，可以使用request.ForceContentType(contentType string)来进行设置解析使用的方式，当然也可以用request.ExpectContentType(contentType string)只不过优先级最小

```go
func parseResponseBody(c *Client, res *Response) (err error) {
	if res.StatusCode() == http.StatusNoContent {
		return
	}
	// Handles only JSON or XML content type
	ct := firstNonEmpty(res.Request.forceContentType, res.Header().Get(hdrContentTypeKey), res.Request.fallbackContentType)    //根据上述顺序进行决定解析的方式
	if IsJSONType(ct) || IsXMLType(ct) {
		// HTTP status code > 199 and < 300, considered as Result
		if res.IsSuccess() {
			res.Request.Error = nil
			if res.Request.Result != nil {
				err = Unmarshalc(c, ct, res.body, res.Request.Result)
				return
			}
		}

		// HTTP status code > 399, considered as Error
		if res.IsError() {
			// global error interface
			if res.Request.Error == nil && c.Error != nil {
				res.Request.Error = reflect.New(c.Error).Interface()
			}

			if res.Request.Error != nil {
				err = Unmarshalc(c, ct, res.body, res.Request.Error)
			}
		}
	}

	return
}
```

反序列化后的内容可以通过response读取

```go
func (r *Response) Result() interface{} {
	return r.Request.Result
}
```

##### 重试功能

 重试相关属性如下

```go
RetryConditionFunc func(*Response, error) bool      //判断是否进行重试的hook

OnRetryFunc func(*Response, error)						//触发重试的hook

RetryAfterFunc func(*Client, *Response) (time.Duration, error) //进一步控制重试时间间隔

Options struct {
		maxRetries      int       //重试最大次数
		waitTime        time.Duration   //重试的时间间隔
		maxWaitTime     time.Duration   //重试最大时间间隔
		retryConditions []RetryConditionFunc    //判断是否重试
		retryHooks      []OnRetryFunc 		//重试时的hook
		resetReaders    bool      
	}
```

相关函数

```go
//设置单个request请求的重试条件
func (r *Request) AddRetryCondition(condition RetryConditionFunc) *Request {
	/*........*/
	return r
}

```

```go
//设置重试次数，默认3
func (c *Client) SetRetryCount(count int) *Client {
	/*........*/
	return c
}

//设置重试等待时间，默认100millisecond
func (c *Client) SetRetryWaitTime(waitTime time.Duration) *Client {
	/*........*/
	return c
}

//设置最大重试等待时间，默认2second
func (c *Client) SetRetryMaxWaitTime(maxWaitTime time.Duration) *Client {
	/*........*/
	return c
}

//用来进一步控制重试时间
func (c *Client) SetRetryAfter(callback RetryAfterFunc) *Client {
	/*........*/
	return c
}

//添加用来控制请求重试条件
func (c *Client) AddRetryCondition(condition RetryConditionFunc) *Client {
	/*........*/
	return c
}

//添加重试时的hook
func (c *Client) AddRetryHook(hook OnRetryFunc) *Client {
	/*........*/
	return c
}
```

##### Trace

用来跟踪http请求的详细时间和连接信息，使用如下函数启用禁用

```go
func (r *Request) EnableTrace() *Request {
  /*........*/
	return r
}

func (c *Client) DisableTrace() *Client {
	/*........*/
	return c
}

func (c *Client) EnableTrace() *Client {
	/*........*/
	return c
}
```

读取trace信息

```go
//返回各个状态持续的时间
func (r *Request) TraceInfo() TraceInfo {
	/*........*/
	return ti
}

// ConnectInfo 返回连接相关时间点和地址信息
func (r *Request) ConnectInfo() ConnectInfo {
	/*........*/
	return ci
}
```

### Response

封装了相关易于读取结果的接口，比较常用的如下

```go
//返回请求的body，他会返回body中的内容，在请求完成自动读取的。但是总所周知，body读完之后就不能在被读了，所以针对这个自动读取有一个开关可以禁用自动读取：func (c *Client) SetDoNotParseResponse(parse bool) *Client和func (r *Request) SetDoNotParseResponse(parse bool) *Request
func (r *Response) Body() []byte
//需要自动解析打开
func (r *Response) String() string
//如果被自动解析了，就无法进行读取了，所以此函数可能会返回nil
func (r *Response) RawBody() io.ReadCloser
//读取响应的所有内容（包括header和body），需要保证自动解析打开
func (r *Response) GetRaw() ([]byte, error)

//返回状态
func (r *Response) StatusCode() int
func (r *Response) Status() string

//返回执行body中的内容反序列化结果，当自动读取body不被禁用的时候才能用
func (r *Response) Result() interface{}
func (r *Response) Error() interface{}

//返回response header
func (r *Response) GetHeaders() http.Header

//返回set-cookie设置的cookie字段
func (r *Response) Cookies() []*http.Cookie

//请求和响应的时间
func (r *Response) Time() time.Duration 


```

