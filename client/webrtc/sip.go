package webrtc

import (
	"fmt"
	"strings"

	"github.com/finove/webrtctest/client"
)

// SIPRegister 注册请求
type SIPRegister struct {
	Request                string                 `json:"request,omitempty"`
	Type                   string                 `json:"type,omitempty"`
	SendRegister           *bool                  `json:"send_register,omitempty"`
	ForceUDP               *bool                  `json:"force_udp,omitempty"`
	ForceTCP               *bool                  `json:"force_tcp,omitempty"`
	Sips                   *bool                  `json:"sips,omitempty"`
	Username               string                 `json:"username,omitempty"`
	Secret                 string                 `json:"secret,omitempty"`
	Ha1Secret              string                 `json:"ha1_secret,omitempty"`
	Authuser               string                 `json:"authuser,omitempty"`
	DisplayName            string                 `json:"display_name,omitempty"`
	UserAgent              string                 `json:"user_agent,omitempty"`
	Proxy                  string                 `json:"proxy,omitempty"`
	OutboundProxy          string                 `json:"outbound_proxy,omitempty"`
	Headers                map[string]interface{} `json:"headers,omitempty"`
	ContactParams          map[string]interface{} `json:"contact_params,omitempty"`
	IncomingHeaderPrefixes []string               `json:"incoming_header_prefixes,omitempty"`
	Refresh                *bool                  `json:"refresh,omitempty"`
	RegisterTTL            *int                   `json:"register_ttl,omitempty"`
	MasterID               int64                  `json:"master_id,omitempty"`
}

func (sr SIPRegister) String() (resp string) {
	resp = fmt.Sprintf("%s %s %s %s", sr.Request, sr.Type, sr.Username, sr.Proxy)
	return
}

func (sr *SIPRegister) AddHeader(key string, value interface{}) {
	if sr.Headers == nil {
		sr.Headers = make(map[string]interface{})
	}
	sr.Headers[key] = value
}

func (sr *SIPRegister) ResetHeaders() {
	sr.Headers = make(map[string]interface{})
}

func (sr *SIPRegister) AsRegister(user, password, domain string, masterID int64) {
	sr.Request = "register"
	sr.ForceUDP = client.Bool(true)
	sr.Username = fmt.Sprintf("sip:%s@%s", user, domain)
	sr.Authuser = user
	sr.DisplayName = user
	sr.Secret = password
	sr.Proxy = fmt.Sprintf("sip:%s", domain)
	if masterID > 0 {
		sr.Type = "helper"
		sr.MasterID = masterID
	} else {
		sr.MasterID = 0
	}
}

func (sr *SIPRegister) AsUnregister() {
	sr.Request = "unregister"
}

type SIPCall struct {
	Request   string                 `json:"request,omitempty"`
	CallID    string                 `json:"call_id,omitempty"`
	URI       string                 `json:"uri,omitempty"`
	ReferID   string                 `json:"refer_id,omitempty"`
	Headers   map[string]interface{} `json:"headers,omitempty"`
	Secret    string                 `json:"secret,omitempty"`
	Ha1Secret string                 `json:"ha1_secret,omitempty"`
	Authuser  string                 `json:"authuser,omitempty"`
	Code      int                    `json:"code,omitempty"`
}

func (sc *SIPCall) AddHeader(key string, value interface{}) {
	if sc.Headers == nil {
		sc.Headers = make(map[string]interface{})
	}
	sc.Headers[key] = value
}

func (sc *SIPCall) Call(uri string) {
	sc.Request = "call"
	sc.URI = uri
}

func (sc *SIPCall) Accept() {
	sc.Request = "accept"
}

func (sc *SIPCall) Decline(code int) {
	sc.Request = "decline"
	sc.Code = code
}

func (sc *SIPCall) Hangup() {
	sc.Request = "hangup"
}

type SIPEventResult struct {
	Event        string                 `json:"event,omitempty"`
	Code         int                    `json:"code,omitempty"`
	Reason       string                 `json:"reason,omitempty"`
	Username     string                 `json:"username,omitempty"`
	Displayname  string                 `json:"displayname,omitempty"`
	Callee       string                 `json:"callee,omitempty"`
	RegisterSent bool                   `json:"register_sent,omitempty"`
	Headers      map[string]interface{} `json:"headers,omitempty"`
	Helper       bool                   `json:"helper,omitempty"`
	MasterID     int64                  `json:"master_id,omitempty"`
}

func (sr SIPEventResult) HeadersFrom() string {
	if v, ok := sr.Headers["from"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (sr SIPEventResult) HeadersTo() string {
	if v, ok := sr.Headers["to"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (sr SIPEventResult) HeadersContactUser() string {
	if v, ok := sr.Headers["contact_user"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// SipEvent sip event
type SipEvent struct {
	Sip       string         `json:"sip,omitempty"`
	Result    SIPEventResult `json:"result,omitempty"`
	CallID    string         `json:"call_id,omitempty"`
	ErrorCode int            `json:"error_code,omitempty"`
	Error     string         `json:"error,omitempty"`
	SDP       string         `json:"sdp,omitempty"`    // from janus resp
	Sender    int64          `json:"sender,omitempty"` // event handle id
}

// SIPHeader get sip header from result
func (se SipEvent) SIPHeader(name string) string {
	if v, ok := se.Result.Headers[name]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// Event event name
func (se SipEvent) Event() string {
	return se.Result.Event
}

// Identify sip call handle identify
func (se SipEvent) Identify() string {
	// if se.Handle != nil {
	// 	return se.Handle.Ident
	// }
	return ""
}

// SIPURI sip call user
func (se SipEvent) SIPURI() string {
	return se.Result.Username
}

// Displayname sip call displayname
func (se SipEvent) Displayname() string {
	var displayName = strings.TrimPrefix(se.Result.Displayname, "\"")
	displayName = strings.TrimSuffix(displayName, "\"")
	return displayName
}

// SIPFrom sip call from
func (se SipEvent) SIPFrom() (ret string) {
	ret = se.Result.HeadersFrom()
	if ret == "" {
		ret = se.Result.Username
	}
	return
}

// SIPTo sip call to
func (se SipEvent) SIPTo() (ret string) {
	ret = se.Result.HeadersTo()
	if ret == "" {
		ret = se.Result.Callee
	}
	return
}

// SIPSdp sip sdp
func (se SipEvent) SIPSdp() string {
	return se.SDP
}

// SIPCallID sip call id
func (se SipEvent) SIPCallID() string {
	return se.CallID
}

// PeerUser sip call peer
func (se SipEvent) PeerUser() string {
	return se.Result.HeadersContactUser()
}

// GetCalleeNumber 获取SIP URI中的号码部分
// "username": "sip:pbx.000ea93d209c-ers6sr@proxy.newlync.com"
// "callee": "sip:652345@proxy.newlync.com"
func GetCalleeNumber(callee string) string {
	var fields []string
	if !strings.HasPrefix(callee, "sip:") {
		return ""
	}
	fields = strings.Split(strings.TrimPrefix(callee, "sip:"), "@")
	if len(fields) != 2 {
		return ""
	}
	return fields[0]
}
