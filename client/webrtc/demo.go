package webrtc

import (
	"fmt"
	"log"
)

// VideoRoomApp video room demo
type VideoRoomApp struct {
	cli     *Client
	session *Janus
	ctl     *Handle
}

// NewVideoRoomApp create new video room app
func NewVideoRoomApp(server, secret string) (app *VideoRoomApp, err error) {
	app = new(VideoRoomApp)
	app.cli = NewClient(server, secret)
	if app.session, err = app.cli.NewJanus(); err != nil {
		return
	}
	app.ctl, err = app.session.Attach(PluginVideoRoom, "control")
	return
}

// Status statics
func (app *VideoRoomApp) Status() (err error) {
	log.Printf("video room app session id %d", app.session.GetSessionID())
	app.session.ShowHandles()
	return
}

// NewParticipant 新的会议参与人
func (app *VideoRoomApp) NewParticipant(roomID int64, display string, pin ...string) (pub *Handle, resp *VideoRoomResponse, err error) {
	var req VideoRoomJoin
	var roomResp VideoRoomResponse
	if pub, err = app.session.Attach(PluginVideoRoom, "publish"); err != nil {
		return
	}
	req.AsPublisher(roomID, display)
	if len(pin) > 0 {
		req.Pin = pin[0]
	}
	if _, err = pub.Send(&req, nil, &roomResp); err != nil {
		err = fmt.Errorf("join room fail:%w", err)
	} else {
		resp = &roomResp
		log.Printf("new participant %d", resp.ID)
	}
	return
}
