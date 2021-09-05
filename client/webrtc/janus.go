package webrtc

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/finove/golibused/pkg/logger"
	"github.com/gorilla/websocket"
)

// support plugin name
const (
	PluginSIP       = "janus.plugin.sip"
	PluginVideoRoom = "janus.plugin.videoroom"
)

// CtxKey context key for save value
type CtxKey int

// context key defined
const (
	CtxClientID CtxKey = iota
	CtxIdentify
	CtxCallSession
	CtxParticipantID
	CtxScreenShare
)

// Client janus client
type Client struct {
	Server string
	Secret string
}

// NewClient 新的janus客户端，配置服务器和密码
func NewClient(server, secret string) (cli *Client) {
	cli = new(Client)
	cli.Server = server
	cli.Secret = secret
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return
}

// NewJanus 创建新的janus会话
func (cli *Client) NewJanus() (js *Janus, err error) {
	var h = http.Header{}
	js = &Janus{
		cli: cli,
	}
	h.Add("Sec-WebSocket-Protocol", "janus-protocol") // janus-admin-protocol
	js.conn, _, err = websocket.DefaultDialer.Dial(js.GetServer(), h)
	if err != nil {
		js = nil
		return
	}
	logger.Info("new session connect to janus webrtc server %s ok", js.GetServer())
	js.disconn = make(chan bool)
	js.idEncode = base32.NewEncoding("ABCDEFGHJKLabcdefghjkmnopqMNWYZz")
	go js.consumeEvent()
	if err = js.newSession(); err != nil {
		js.Destroy()
		js = nil
		return
	}
	go func() {
	OUTLOOP:
		for {
			select {
			case <-js.disconn:
				js.Destroy()
				break OUTLOOP
			case <-time.After(30 * time.Second):
				if js.GetSessionID() > 0 {
					js.keepAlive()
				}
			}
		}
		logger.Info("session %d keep alive finish", js.GetSessionID())
		if js.callback != nil {
			go js.callback(js, "session finish")
		}
	}()
	return
}

// Janus session
type Janus struct {
	cli      *Client
	id       int64 // session id
	wlock    sync.Mutex
	disconn  chan bool
	callback func(*Janus, string)
	conn     *websocket.Conn
	idEncode *base32.Encoding
	handles  sync.Map
	waitEv   sync.Map
}

// GetServer 返回服务器地址
func (js *Janus) GetServer() string {
	return js.cli.Server
}

// IsConnected 是否连接到服务器
func (js *Janus) IsConnected() bool {
	return js.conn != nil
}

// GetSessionID 获取会话ID
func (js *Janus) GetSessionID() int64 {
	return js.id
}

// Attach 绑定插件，创建handle
func (js *Janus) Attach(pluginName, tag string) (h *Handle, err error) {
	var req janusRequest
	var resp *JanusResponse
	req.Janus = "attach"
	req.APISecret = js.cli.Secret
	req.SessionID = js.id
	req.Plugin = pluginName
	resp, err = js.requestWait(&req)
	if err != nil {
		err = fmt.Errorf("janus plugin attach %s fail:%w", pluginName, err)
		return
	}
	if err = resp.HasError(); err != nil {
		return
	}
	h = &Handle{
		js:     js,
		plugin: pluginName,
		tag:    tag,
		Status: "init",
	}
	h.ID = resp.dataID()
	h.Ctx = context.Background()
	js.handles.Store(h.ID, h)
	logger.Info("got %s %s handle ID %d", tag, pluginName, h.ID)
	return
}

// Destroy 释放会话，断开连接
func (js *Janus) Destroy() (err error) {
	if js.conn == nil {
		return
	}
	defer func() {
		js.conn.Close()
		js.conn = nil
	}()
	js.wlock.Lock()
	err = js.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	js.wlock.Unlock()
	return
}

// SetEventCallBack 设置事件处理回调
func (js *Janus) SetEventCallBack(f func(*Janus, string)) *Janus {
	if f != nil {
		js.callback = f
	}
	return js
}

// ShowHandles 输出当前会话中所有handle信息
func (js *Janus) ShowHandles() {
	js.handles.Range(func(key interface{}, value interface{}) bool {
		if h, ok := value.(*Handle); ok {
			logger.Info("%d handle %s", js.id, h.Summary())
		}
		return true
	})
}

func (js *Janus) keepAlive() {
	var req janusRequest
	if js.conn == nil {
		return
	}
	req.Janus = "keepalive"
	req.APISecret = js.cli.Secret
	req.SessionID = js.id
	js.requestWait(&req)
}

func (js *Janus) newSession() (err error) {
	var req janusRequest
	var resp *JanusResponse
	req.Janus = "create"
	req.APISecret = js.cli.Secret
	resp, err = js.requestWait(&req)
	if err != nil {
		err = fmt.Errorf("newSession create fail:%w", err)
		return
	}
	js.id = resp.dataID()
	logger.Info("new janus session ID %d", js.id)
	return
}

func (js *Janus) requestWait(req *janusRequest, timeOuts ...time.Duration) (resp *JanusResponse, err error) {
	var timeOut time.Duration = 5 * time.Second
	var ev = newEventAck(req.Transaction)
	if js.conn == nil {
		err = fmt.Errorf("no connection to the server")
		return
	}
	if len(timeOuts) > 0 {
		timeOut = timeOuts[0]
	}
	if req.Transaction == "" {
		req.Transaction = js.newTransactionID()
	}
	js.waitEv.Store(req.Transaction, ev)
	defer js.waitEv.Delete(req.Transaction)
	js.wlock.Lock()
	err = js.conn.WriteJSON(req)
	js.wlock.Unlock()
	if req.Janus != "keepalive" {
		logger.Info("wait request %s", req)
	}
	select {
	case <-ev.done:
		resp = ev.value
	case <-time.After(timeOut):
		err = fmt.Errorf("request timeout with transaction %s", req.Transaction)
	}
	return
}

func (js *Janus) consumeEvent() {
	var err error
	var evJanus = map[string]bool{
		"event":     true,
		"webrtcup":  true,
		"dataready": true,
		"media":     true,
		"hangup":    true,
		"detached":  true,
		"slowlink":  true,
	}
	if js.conn == nil {
		logger.Error("consumeEvent no connection to the server")
		return
	}
	for {
		var message []byte
		var notify JanusResponse
		_, message, err = js.conn.ReadMessage()
		if err != nil {
			logger.Warning("read websocket message fail:%v", err)
			break
		}
		err = json.Unmarshal(message, &notify)
		if err != nil {
			continue
		}
		notify.oriMsg = message
		if v, ok := evJanus[notify.Janus]; ok && v {
			go js.processEvent(&notify)
		} else if notify.Transaction != "" {
			// 响应消息按序处理，不在 go 线程中处理
			if value, ok := js.waitEv.Load(notify.Transaction); ok {
				if ea, ok := value.(*eventAck); ok {
					// 异步接口会先发一个 ack 确认消息，然后再发一个 event 响应处理结果
					// 也有可能先发一个 event 响应处理结果，然后再发一个 ack 确认消息，异步关系导致次序不一定
					// logger.Info("janus request %s %s", notify.Transaction, notify.Janus)
					ea.value = &notify
					close(ea.done)
				}
				continue
			}
		} else {
			logger.Error("should not happen, got message from session:%s", string(message))
		}
	}
	js.disconn <- true
	logger.Warning("session %d consumeEvent finish %v", js.GetSessionID(), err)
}

func (js *Janus) processEvent(event *JanusResponse) {
	var value interface{}
	var h *Handle
	var ok bool
	// logger.Info("origin message %s", string(event.oriMsg))
	if event.Sender != 0 {
		if value, ok = js.handles.Load(event.Sender); ok {
			if h, ok = value.(*Handle); ok {
				h.processEvent(event)
			}
		}
		return
	}
	logger.Info("session %d get event from handle %d, event %s", js.id, event.Sender, string(event.oriMsg))
}

func (js *Janus) newTransactionID() (key string) {
	var tmpBuff = make([]byte, 10)
	if n, err := rand.Read(tmpBuff); err != nil || n != 10 {
		logger.Warning("generator transaction id %d, %v", n, err)
	}
	key = js.idEncode.EncodeToString(tmpBuff[:])
	return
}

// Handle janus handle
type Handle struct {
	js         *Janus
	plugin     string
	tag        string
	ID         int64
	Status     string
	Ctx        context.Context `json:"-"`
	callBack   func(*Handle, string, interface{})
	asyncQueue sync.Map // 异步响应队列
	webrtcUp   bool
	dataReady  bool
	// iceState   bool
	// mediaState bool
	// slowLink   bool
}

// GetID 获取handle ID
func (h *Handle) GetID() int64 {
	return h.ID
}

// GetTag 获取handle tag
func (h *Handle) GetTag() string {
	return h.tag
}

// GetPlugin 获取绑定的插件名
func (h *Handle) GetPlugin() string {
	return h.plugin
}

func (h *Handle) DataReady() (yes bool) {
	return h.dataReady
}

// ContextString 获取上下文字段值
func (h *Handle) ContextString(key interface{}) (value string) {
	var ok bool
	if v := h.Ctx.Value(key); v != nil {
		if value, ok = v.(string); !ok {
			value = fmt.Sprintf("%v", v)
		}
	}
	return
}

// ContextInt64 获取上下文字段值
func (h *Handle) ContextInt64(key interface{}) (value int64) {
	var ok bool
	if v := h.Ctx.Value(key); v != nil {
		if value, ok = v.(int64); !ok {
			value1 := fmt.Sprintf("%v", v)
			value, _ = strconv.ParseInt(value1, 0, 64)
		}
	}
	return
}

// Send 发送消息给插件
func (h *Handle) Send(reqBody interface{}, jsep *Jsep, pluginResp ...interface{}) (resp *JanusResponse, err error) {
	var req janusRequest
	req.Janus = "message"
	req.SessionID = h.js.GetSessionID()
	req.HandleID = h.GetID()
	req.APISecret = h.js.cli.Secret
	req.Body = reqBody
	req.Jsep = jsep
	req.Transaction = h.js.newTransactionID()
	ea := newEventAck(req.Transaction)
	h.asyncQueue.Store(req.Transaction, ea)
	resp, err = h.js.requestWait(&req)
	if err != nil {
		err = fmt.Errorf("handle %s send message fail:%w", h.tag, err)
		return
	}
	if resp.Janus == "ack" {
		// wait event
		select {
		case <-ea.done:
			resp = ea.value
		case <-time.After(15 * time.Second):
			err = fmt.Errorf("send wait response timeout with transaction %s", req.Transaction)
		}
	}
	if err != nil {
		return
	}
	if err = resp.HasError(); err != nil {
		return
	}
	if len(pluginResp) > 0 && resp.PluginData != nil && resp.PluginData.Plugin == h.plugin {
		json.Unmarshal(resp.PluginData.Data, pluginResp[0])
	}
	return
}

// Dtmf 发送DTMF tone
func (h *Handle) Dtmf() (err error) {
	err = fmt.Errorf("not implemented")
	return
}

// Data 通过数据通道发送数据
func (h *Handle) Data(data interface{}) (err error) {
	if h.plugin == PluginVideoRoom {
		var req VideoRoomRelayData
		req.SetPayload(data)
		_, err = h.Send(&req, nil)
	} else {
		err = fmt.Errorf("not implemented")
	}
	return
}

// Hangup 挂断连接
func (h *Handle) Hangup() (err error) {
	err = fmt.Errorf("not implemented")
	return
}

// Detach 解绑handle，释放
func (h *Handle) Detach() (err error) {
	var req janusRequest
	var resp *JanusResponse
	req.Janus = "detach"
	req.SessionID = h.js.GetSessionID()
	req.HandleID = h.GetID()
	req.APISecret = h.js.cli.Secret
	req.Transaction = h.js.newTransactionID()
	ea := newEventAck(req.Transaction)
	h.asyncQueue.Store(req.Transaction, ea)
	resp, err = h.js.requestWait(&req)
	if err != nil {
		err = fmt.Errorf("janus plugin detach %s fail:%w", h.plugin, err)
		return
	}
	if resp.Janus == "ack" {
		// wait event
		select {
		case <-ea.done:
			resp = ea.value
		case <-time.After(15 * time.Second):
			err = fmt.Errorf("send wait response timeout with transaction %s", req.Transaction)
		}
	}
	if err != nil {
		return
	}
	if err = resp.HasError("detach"); err == nil {
		h.js.handles.Delete(h.GetID())
	}
	return
}

// SetTag 设置标签
func (h *Handle) SetTag(tag string) *Handle {
	h.tag = tag
	return h
}

// SetStatus 设置状态
func (h *Handle) SetStatus(st string) *Handle {
	h.Status = st
	return h
}

// SetContext 设置状态
func (h *Handle) SetContext(key, val interface{}) *Handle {
	h.Ctx = context.WithValue(h.Ctx, key, val)
	return h
}

// SetEventCallBack 设置事件处理回调
func (h *Handle) SetEventCallBack(f func(*Handle, string, interface{})) *Handle {
	if f != nil {
		h.callBack = f
	}
	return h
}

// Summary handle summary
func (h *Handle) Summary() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("tag %s", h.tag))
	out.WriteString(fmt.Sprintf(" plugin %s", h.plugin))
	out.WriteString(fmt.Sprintf(" ID %d", h.ID))
	out.WriteString(fmt.Sprintf(" status %s", h.Status))
	return out.String()
}

// ShowSiblings 显示兄弟handle信息
func (h *Handle) ShowSiblings() {
	h.js.ShowHandles()
}

func (h *Handle) onMessage(data json.RawMessage, jsep *Jsep) {
	if h.plugin == PluginVideoRoom {
		var roomEvent VideoRoomResponse
		json.Unmarshal(data, &roomEvent)
		switch roomEvent.VideoRoom {
		case "incoming-data":
			h.onData(roomEvent.Data)
		case "talking", "stopped-talking", "active-speaker":
			logger.Info("handle %s(%d) get event %s room %d, id %d, level %v", h.tag, h.GetID(), roomEvent.VideoRoom, roomEvent.Room, roomEvent.ID, roomEvent.AudioLevelAvg)
		case "joined":
			logger.Info("handle %s(%d) %s room %d(%s) with participant id %d", h.tag, h.GetID(), roomEvent.VideoRoom, roomEvent.Room, roomEvent.Description, roomEvent.ID)
			h.Ctx = context.WithValue(h.Ctx, CtxParticipantID, roomEvent.ID)
		case "slow_link":
			logger.Info("handle %s(%d) get event %s current-bitrate %d", h.tag, h.GetID(), roomEvent.VideoRoom, roomEvent.CurrentBitrate)
		case "event":
			logger.Info("handle %s(%d) get event %s=%s with jsep %s", h.tag, h.GetID(), roomEvent.VideoRoom, string(data), jsep.Type)
		case "dataready":
			h.dataReady = true
		default:
			logger.Info("handle %s(%d) get event %s=%s", h.tag, h.GetID(), roomEvent.VideoRoom, string(data))
		}
		if h.callBack != nil {
			h.callBack(h, roomEvent.VideoRoom, &roomEvent)
		}
	} else if h.plugin == PluginSIP {
		var sipEvent SipEvent
		json.Unmarshal(data, &sipEvent)
		if jsep != nil {
			sipEvent.SDP = jsep.SDP
		}
		switch sipEvent.Sip {
		case "event":
		default:
			logger.Info("handle %s(%d) get sip event %s=%s", h.tag, h.GetID(), sipEvent.Sip, string(data))
		}
		if h.callBack != nil {
			h.callBack(h, sipEvent.Sip, &sipEvent)
		}
	}
}

func (h *Handle) onData(data json.RawMessage) {
	logger.Info("handle %s(%d) onData %s", h.tag, h.GetID(), string(data))
}

func (h *Handle) processEvent(event *JanusResponse) {
	if event.Transaction != "" {
		if value, ok := h.asyncQueue.Load(event.Transaction); ok {
			if ea, ok := value.(*eventAck); ok {
				ea.value = event
				close(ea.done)
			}
			h.asyncQueue.Delete(event.Transaction)
			// return
		}
	}
	switch event.Janus {
	case "event":
		if event.PluginData != nil && event.PluginData.Plugin == h.plugin && event.PluginData.Data != nil {
			h.onMessage(event.PluginData.Data, &event.Jsep)
		}
	case "slowlink":
		logger.Info("handle %s(%d) get event %s uplink %v media %s lost %d", h.tag, h.GetID(), event.Janus, event.Uplink, event.Media, event.Lost)
	case "media":
		logger.Info("handle %s(%d) get event %s %s receiving %v", h.tag, h.GetID(), event.Janus, event.Type, event.Receiving)
	case "hangup":
		logger.Info("handle %s(%d) get event %s reason %s", h.tag, h.GetID(), event.Janus, event.Reason)
		if h.callBack != nil {
			h.callBack(h, event.Janus, nil)
		}
		h.dataReady = false
		h.webrtcUp = false
	case "detached":
		logger.Info("handle %s(%d) get event %s", h.tag, h.GetID(), event.Janus)
	case "dataready":
		h.dataReady = true
		fallthrough
	case "webrtcup":
		logger.Info("handle %s(%d) get event %s", h.tag, h.GetID(), event.Janus)
		if event.Janus == "webrtcup" {
			h.webrtcUp = true
		}
		if h.callBack != nil {
			h.callBack(h, event.Janus, nil)
		}
	default:
		logger.Info("handle %s(%d) get event %s", h.tag, h.GetID(), string(event.oriMsg))
	}
}
