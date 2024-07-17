package httpc

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"
)

func TestRequest_ConnectInfo(t *testing.T) {
	client := dc()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).SetTimeout(time.Second * 3).GetLocalAddrBySelf("").DisableRedirect()
	resp, err := client.R().EnableTrace().Get("http://192.168.105.112/aaaaa")
	if err != nil {
		fmt.Println(err)
	}
	cr := resp.Request.ConnectInfo()
	a, _ := resp.Request.GetRaw()
	b, _ := resp.GetRaw()
	fmt.Println(string(a))
	fmt.Println(string(b))
	fmt.Println("准备建立连接时间：", cr.GetConnectTime)
	fmt.Println("成功建立连接时间/发送请求数据时间：", cr.GotConnectTime)
	fmt.Println("接收到返回包时间：", cr.ReceiveHttpResponseTime)
	fmt.Println("连接结束时间：", cr.ConnectDone)
	fmt.Println("远程地址：", cr.RemoteAddr)
	fmt.Println("本地地址：", cr.LocalAddr)
}

func TestSaveEventToLocalFile_Push(t *testing.T) {
	c := NewDefaultClient()
	save, err := NewSaveEventToLocalFile("t.txt")
	if err != nil {
		panic(err)
	}
	defer save.Close()
	save.SetLog(c.log)
	c.AppendPushEvent(save)
	target := "http://192.168.105.112"
	c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).SetLogger(createLogger())
	_, err = c.R().EnableTrace().SetIndex(target).SetQueryParam("a", "asdads").Get(target)
	if err != nil {
		fmt.Println(err)
	}
}

func send(c *Client) {
	target := "https://192.168.101.232"
	c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).SetProxy("http://127.0.0.1:8080").SetLogger(createLogger())
	var err error
	go func() {
		for i := 0; i < 10; i++ {
			_, err = c.R().EnableTrace().SetIndex(target).Get(target)
			if err != nil {
				panic(err)
			}
		}
	}()
	go func() {
		for i := 0; i < 10; i++ {
			_, err = c.R().EnableTrace().SetIndex(target).Get(target)
			if err != nil {
				panic(err)
			}
		}
	}()

	time.Sleep(time.Second * 5)
}

func TestRequest_SetBody(t *testing.T) {
	c := NewDefaultClient().SetProxy("http://127.0.0.1:8080")
	req := c.NewRequest().SetBody("asdasdasdasd")
	req.Post("http://127.0.0.1:7001")
	b, _ := req.GetBody()
	fmt.Println(string(b))

}
