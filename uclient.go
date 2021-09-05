package main

import (
	"fmt"
	"log"

	"github.com/finove/webrtctest/client"
	jns "github.com/finove/webrtctest/client/webrtc"
)

var (
	janusAddress = "ws://172.28.128.108:8188"
	janusSecret  = "janusrocks"
	// janusAdminkey = "supersecret"
)

type uClient struct {
	janusCli *jns.Client
	session  *jns.Janus
	roomCtl  *jns.Handle
	sub      *jns.Handle
	pub      *jns.Handle
}

func (uc *uClient) Init(server, secret string) (err error) {
	uc.janusCli = jns.NewClient(server, secret)
	if uc.session, err = uc.janusCli.NewJanus(); err != nil {
		return
	}
	uc.roomCtl, err = uc.session.Attach(jns.PluginVideoRoom, "control")
	return
}

func (uc *uClient) listparticipants(roomID int64) (err error) {
	var req jns.VideoRoomCommon
	var roomResp jns.VideoRoomResponse
	req.Request = "listparticipants"
	req.Room = roomID
	if _, err = uc.roomCtl.Send(&req, nil, &roomResp); err != nil {
		return
	}
	log.Printf("%s", client.ShowJSON(roomResp, true))
	return
}

func (uc *uClient) Subscrite(roomID, feedID int64) (sub *jns.Handle, offer string, err error) {
	var req jns.VideoRoomJoin
	var resp *jns.JanusResponse
	var roomResp jns.VideoRoomResponse
	if sub, err = uc.session.Attach(jns.PluginVideoRoom, "subscriber"); err != nil {
		return
	}
	req.AsSubscriber(roomID, feedID)
	if resp, err = sub.Send(&req, nil, &roomResp); err != nil {
		err = fmt.Errorf("subscribe fail:%w", err)
	} else {
		if resp.Jsep.SDP != "" && resp.Jsep.Type == "offer" {
			offer = resp.Jsep.SDP
			uc.sub = sub
		}
	}
	return
}

func (uc *uClient) SubscriteStart(sdp string) (err error) {
	var roomResp jns.VideoRoomResponse
	var req struct {
		Request string `jsno:"request"`
	}
	var jsep = new(jns.Jsep)
	jsep.Type = "answer"
	jsep.SDP = sdp
	jsep.Trickle = client.Bool(false)
	req.Request = "start"
	if _, err = uc.sub.Send(&req, jsep, &roomResp); err != nil {
		err = fmt.Errorf("subscribe start fail:%w", err)
	}
	return
}

func (uc *uClient) JoinRoom(roomID int64) (err error) {
	var req jns.VideoRoomJoin
	if uc.pub, err = uc.session.Attach(jns.PluginVideoRoom, "publish"); err != nil {
		return
	}
	req.AsPublisher(roomID, "webtest")
	_, err = uc.pub.Send(&req, nil)
	return
}

func (uc *uClient) Publish(sdp string) (answer string, err error) {
	var req jns.VideoRoomPublish
	var resp *jns.JanusResponse
	var roomResp jns.VideoRoomResponse
	var jsep = new(jns.Jsep)
	jsep.Type = "offer"
	jsep.SDP = sdp
	req.SetupInit("webtest")
	if resp, err = uc.pub.Send(&req, jsep, &roomResp); err != nil {
		return
	}
	if resp.Jsep.SDP != "" && resp.Jsep.Type == "answer" {
		answer = resp.Jsep.SDP
	}
	return
}
