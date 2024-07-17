package httpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

const l1 = "2006-01-02 15:04:05"

// PushTraceEvent 推送事件接口
type PushTraceEvent interface {
	// Push 根据索引将请求响应事件内容推送到其他地方
	Push(index interface{}, event TraceEvent)
}

// TraceEvent 可以跟踪client的每个请求的请求事件和响应事件，
type TraceEvent struct {
	ConnectInfo ConnectInfo `json:"connectInfo"`
	RequestRaw  []byte      `json:"requestRaw"`
	ResponseRaw []byte      `json:"responseRaw"`
	Error       string      `json:"error"`
}

func (t TraceEvent) String() string {
	var re string
	re += fmt.Sprintf("准备建立连接时间:%v\n", t.ConnectInfo.GetConnectTime.Format(l1))
	re += fmt.Sprintf("成功建立连接时间/发送请求数据时间:%v\n", t.ConnectInfo.GotConnectTime.Format(l1))
	re += fmt.Sprintf("接收到返回包时间:%v\n", t.ConnectInfo.ReceiveHttpResponseTime.Format(l1))
	re += fmt.Sprintf("连接结束时间:%v\n", t.ConnectInfo.ConnectDone.Format(l1))
	re += fmt.Sprintf("本地地址:%v\n", t.ConnectInfo.LocalAddr)
	re += fmt.Sprintf("远程地址:%v\n\n", t.ConnectInfo.RemoteAddr)
	re += fmt.Sprintf("请求数据包:\n\t%v\n", strings.ReplaceAll(string(t.RequestRaw), "\n", "\n\t"))
	re += fmt.Sprintf("响应数据包:\n\t%v\n", strings.ReplaceAll(string(t.ResponseRaw), "\n", "\n\t"))
	if t.Error != "" {
		re += fmt.Sprintf("Error:%v\n", t.Error)
	}
	return re
}

type TraceEventWithIndex struct {
	TraceEvent
	Index interface{} `json:"index"`
}

// NewTraceEvent 新建请求事件
func NewTraceEvent(req *Request, resp *Response, err string) TraceEvent {
	event := new(TraceEvent)
	var reqRawErr error
	var respRawErr error
	event.ConnectInfo = req.ConnectInfo()
	if err != "" {
		event.Error = err
	}
	if req != nil {
		event.RequestRaw, reqRawErr = req.GetRaw()
		if reqRawErr != nil {
			req.log.Errorf("get request raw error:%v", reqRawErr)
		}
	}
	if resp != nil {
		event.ResponseRaw, respRawErr = resp.GetRaw()
		if respRawErr != nil {
			req.log.Errorf("get response raw error:%v", respRawErr)
		}
	}
	return *event
}

type SaveEventToLocalFile struct {
	Filename string
	log      Logger
	file     *os.File
	lock     sync.RWMutex
}

func NewSaveEventToLocalFile(filename string) (*SaveEventToLocalFile, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	save := new(SaveEventToLocalFile)
	save.Filename = filename
	save.file = file
	return save, nil
}
func (s *SaveEventToLocalFile) SetLog(log Logger) {
	s.log = log
}

func (s *SaveEventToLocalFile) Push(index interface{}, event TraceEvent) {
	s.lock.Lock()
	s.writeToFile(TraceEventWithIndex{Index: index, TraceEvent: event})
	s.lock.Unlock()
}

func (s *SaveEventToLocalFile) writeToFile(data TraceEventWithIndex) {
	writeData := "Index:" + fmt.Sprintf("%v", data.Index) + "\n\n" + data.TraceEvent.String() + "========================================================================================================\n"
	_, err := s.file.Write([]byte(writeData))
	if err != nil {
		s.log.Errorf("Write error:%v", err)
	}
}

// Close 关闭文件
func (s *SaveEventToLocalFile) Close() {
	_ = s.file.Close()
}

type PushEventToRemoteAddr struct {
	log  log.Logger
	lock sync.RWMutex
	conn net.Conn
}

func NewPushEventToRemoteAddr(addr string) (*PushEventToRemoteAddr, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	push := new(PushEventToRemoteAddr)
	push.conn = conn
	push.lock = sync.RWMutex{}
	return push, nil
}

func (p *PushEventToRemoteAddr) Push(index interface{}, event TraceEvent) {
	p.lock.Lock()
	p.send(TraceEventWithIndex{Index: index, TraceEvent: event})
	p.lock.Unlock()
}

func (p *PushEventToRemoteAddr) send(data TraceEventWithIndex) {
	//bas使用的推送函数，在bas场景下需要过滤掉一部分错误
	if strings.Contains(data.Error, "auto redirect is disabled") {
		data.Error = ""
	}
	d, err := json.Marshal(&data)
	if err != nil {
		panic(fmt.Sprintf("push request info marshal error:%v,index:%v,data:%v", err, data.Index, data.TraceEvent))
	}
	_, err = p.conn.Write(append(d, []byte("\n")...))
	if err != nil {
		panic(fmt.Sprintf("push request info write to connect error:%v", err))
	}
}

func (p *PushEventToRemoteAddr) Close() {
	_ = p.conn.Close()
}
