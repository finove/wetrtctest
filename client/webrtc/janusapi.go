package webrtc

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/finove/webrtctest/client"
)

type Error interface {
	error
	Code() int
	Reason() string
}

type eventAck struct {
	done  chan struct{}  // 响应到达
	id    string         // transaction id
	value *JanusResponse // ack struct
	// ts    time.Time      // 响应时间
}

func newEventAck(id string) *eventAck {
	return &eventAck{
		done: make(chan struct{}),
		id:   id,
	}
}

type janusRequest struct {
	Janus       string      `json:"janus,omitempty"`
	Transaction string      `json:"transaction,omitempty"`
	APISecret   string      `json:"apisecret,omitempty"`
	SessionID   int64       `json:"session_id,omitempty"`
	HandleID    int64       `json:"handle_id,omitempty"`
	Plugin      string      `json:"plugin,omitempty"`
	Body        interface{} `json:"body,omitempty"`
	Jsep        *Jsep       `json:"jsep,omitempty"`
}

// JSON json show
func (jr janusRequest) JSON() (out string) {
	return client.ShowJSON(&jr)
}

func (jr janusRequest) String() (out string) {
	var fields []string
	fields = append(fields, fmt.Sprintf("janus %s", jr.Janus))
	fields = append(fields, fmt.Sprintf("transaction %s", jr.Transaction))
	if jr.SessionID != 0 {
		fields = append(fields, fmt.Sprintf("session %d", jr.SessionID))
	}
	if jr.HandleID != 0 {
		fields = append(fields, fmt.Sprintf("handle %d", jr.HandleID))
	}
	if jr.Janus == "attach" {
		fields = append(fields, fmt.Sprintf("plugin %s", jr.Plugin))
	} else if jr.Janus == "message" {
		body := jr.Body
		if reg, ok := body.(SIPRegister); ok {
			if reg.Type == "helper" {
				fields = append(fields, " register helper")
			} else {
				vv := client.ShowJSON(body, true)
				fields = append(fields, string(vv))
			}
		} else {
			vv := client.ShowJSON(body, true)
			fields = append(fields, "body "+string(vv))
		}
	}
	if jr.Jsep != nil {
		vv := client.ShowJSON(jr.Jsep, true)
		fields = append(fields, "Jsep "+string(vv))
	}
	out = strings.Join(fields, ",")
	return
}

// JanusResponse janus response
type JanusResponse struct {
	Janus       string          `json:"janus,omitempty"`
	SessionID   int64           `json:"session_id,omitempty"` // session id
	Sender      int64           `json:"sender,omitempty"`     // handle id
	Reason      string          `json:"reason,omitempty"`
	Transaction string          `json:"transaction,omitempty"`
	Media       string          `json:"media,omitempty"`
	Type        string          `json:"type,omitempty"`
	Receiving   bool            `json:"receiving,omitempty"`
	Uplink      bool            `json:"uplink,omitempty"`
	Lost        int             `json:"lost,omitempty"`
	PluginData  *janusPlugin    `json:"plugindata,omitempty"` // a JSON object containing the info coming from the plugin itself
	Data        json.RawMessage `json:"data,omitempty"`
	Jsep        Jsep            `json:"jsep,omitempty"`
	Error       *respError      `json:"error,omitempty"`
	oriMsg      []byte
}

func (resp *JanusResponse) dataID() (id int64) {
	var d struct {
		ID int64 `json:"id"`
	}
	if resp.Data != nil {
		json.Unmarshal(resp.Data, &d)
	}
	id = d.ID
	return
}

// HasError 返回响应的错误
func (resp *JanusResponse) HasError(infos ...string) (err error) {
	if resp.Janus == "error" {
		if resp.Error != nil {
			err = NewError(resp.Error.Code, resp.Error.Reason, infos...)
		} else {
			err = NewError(-1, "unknown", infos...)
		}
	} else if resp.Janus == "event" && resp.PluginData != nil && resp.PluginData.Data != nil {
		var pluginError PluginRespError
		if json.Unmarshal(resp.PluginData.Data, &pluginError) == nil && pluginError.InErrorCode != 0 {
			err = NewError(pluginError.InErrorCode, pluginError.InError, infos...)
		}
	}
	return
}

type respError struct {
	Code   int    `json:"code,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// janusPlugin a JSON object containing the info coming from the plugin itself
type janusPlugin struct {
	Plugin string          `json:"plugin,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type PluginRespError struct {
	Info        string
	InErrorCode int    `json:"error_code"`
	InError     string `json:"error"`
}

func (pre *PluginRespError) Error() string {
	return fmt.Sprintf("%s error code %d,reason %s", pre.Info, pre.InErrorCode, pre.InError)
}

func (pre *PluginRespError) Code() int {
	return pre.InErrorCode
}

func (pre *PluginRespError) Reason() string {
	return pre.InError
}

func NewError(code int, reason string, infos ...string) error {
	return &PluginRespError{Info: strings.Join(infos, " "), InErrorCode: code, InError: reason}
}

// Jsep janus sdp
type Jsep struct {
	Type    string `json:"type,omitempty"`
	SDP     string `json:"sdp,omitempty"`
	Update  bool   `json:"update,omitempty"`
	Trickle *bool  `json:"trickle,omitempty"`
}
