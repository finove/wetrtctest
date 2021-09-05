package client

// MeetingEvent 会议事件
type MeetingEvent interface {
	Event() string                                     // 事件类型，如 join,leave,publish,subscribe,offer,answer
	Conversation() string                              // newlync conversation
	Identity() string                                  // newlync机器人 userID.clientID
	PeerIdentity() string                              // newlync 会议参与者 userID.clientID
	Displayname() string                               // 会议参与者名字
	CallProps(name string) (value string, exists bool) // 获取视频呼叫属性
	Sdp() string                                       // sdp
	CallSessID() string                                // newlync call sessid
}

// MeetingBotEvent 会议机器人事件
type MeetingBotEvent interface {
	MeetingEvent
	RoomID() int64
	MeetID() string
	RoomPin() string
	RoomIDFrom() string
	VirtualUserID() string
}

type CallBotEvent interface {
	MeetingEvent
	CallerNum() string // 主叫号码
	CalleeNum() string // 被叫号码
	VirtualUserID() string
}
