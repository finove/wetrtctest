package webrtc

import (
	"encoding/json"

	"github.com/finove/webrtctest/client"
)

// VideoRoomResponse 视频会议插件事件响应，各种响应定义放到一起
type VideoRoomResponse struct {
	PluginRespError
	VideoRoom      string             `json:"videoroom,omitempty"` // created,edited,destroyed
	Room           int64              `json:"room"`
	Permanent      bool               `json:"permanent"`
	Exists         bool               `json:"exists"`
	CurrentBitrate int                `json:"current-bitrate,omitempty"` // for slow_link
	Unpublished    int64              `json:"unpublished,omitempty"`
	Leaving        int64              `json:"leaving,omitempty"`
	Description    string             `json:"description,omitempty"`
	ID             int64              `json:"id,omitempty"`
	PrivateID      int64              `json:"private_id,omitempty"`
	Allowed        []string           `json:"allowed,omitempty"`
	List           []RoomInfo         `json:"list,omitempty"`
	Participants   []ParticipantsInfo `json:"participants,omitempty"`
	Publishers     []VideoPublisher   `json:"publishers,omitempty"`
	Attendees      []VideoPublisher   `json:"attendees,omitempty"` // only id and display
	Switched       string             `json:"switched,omitempty"`
	Data           json.RawMessage    `json:"data,omitempty"`
	AudioLevelAvg  float64            `json:"audio-level-dBov-avg,omitempty"`
	RelayData      string             `json:"relay_data,omitempty"`
}

// VideoRoomCreate 创建视频会议室请求
type VideoRoomCreate struct {
	Request            string   `json:"request,omitempty"`
	Room               int64    `json:"room,omitempty"`
	Permanent          bool     `json:"permanent"`
	Description        string   `json:"description,omitempty"`
	Publishers         *int     `json:"publishers,omitempty"`           // max number of concurrent senders, default = 3
	Secret             string   `json:"secret,omitempty"`               // password required to edit/destroy the room, optional
	Pin                string   `json:"pin,omitempty"`                  // password required to join the room, optional
	IsPrivate          bool     `json:"is_private"`                     // whether the room should appear in a list request
	Allowed            []string `json:"allowed,omitempty"`              // array of string tokens users can use to join this room, optional
	AdminKey           string   `json:"admin_key,omitempty"`            // 插件配置中有 admin_key 时，只有正确的 key 才能创建会议室
	AudiolevelEvent    *bool    `json:"audiolevel_event,omitempty"`     // (whether to emit event to other users or not, default=false)
	AudioLevelAverage  *int     `json:"audio_level_average,omitempty"`  // (average value of audio level, 127=muted, 0='too loud', default=25)
	AudioActivePackets *int     `json:"audio_active_packets,omitempty"` // (number of packets with audio level, default=100, 2 seconds)
	OpusFec            *bool    `json:"opus_fec,omitempty"`             // whether inband FEC must be negotiated; only works for Opus, default=false
	AudioCodec         string   `json:"audiocodec,omitempty"`           // opus|g722|pcmu|pcma|isac32|isac16
	VideoCodec         string   `json:"videocodec,omitempty"`           // vp8|vp9|h264|av1|h265
}

// Prepare 准备请求
func (vrc *VideoRoomCreate) Prepare(adminKey string, permanent bool, desc, secret, pin string) {
	vrc.Request = "create"
	vrc.AdminKey = adminKey
	vrc.Permanent = permanent
	vrc.Description = desc
	vrc.Secret = secret
	vrc.Pin = pin
	vrc.IsPrivate = false
	vrc.OpusFec = client.Bool(true)
	vrc.AudioCodec = "opus,pcma,pcmu"
	vrc.AudiolevelEvent = client.Bool(true)
	vrc.AudioActivePackets = client.Int(100)
	vrc.AudioLevelAverage = client.Int(50)
}

// VideoRoomEdit 修改视频会议室配置，只支持部分属性配置
type VideoRoomEdit struct {
	Request        string `json:"request"`
	Room           int64  `json:"room"`
	Secret         string `json:"secret,omitempty"`
	NewDescription string `json:"new_description,omitempty"`
	NewSecret      string `json:"new_secret,omitempty"`
	NewPin         string `json:"new_pin,omitempty"`
	NewIsPrivate   bool   `json:"new_is_private,omitempty"`
	Permanent      bool   `json:"permanent"`
	// more ...
}

// VideoRoomDestroy can be used to destroy an existing video room, whether created dynamically or statically
type VideoRoomDestroy struct {
	Request   string `json:"request"`
	Room      int64  `json:"room"`
	Secret    string `json:"secret,omitempty"`
	Permanent bool   `json:"permanent"`
}

// Prepare 准备请求
func (vrd *VideoRoomDestroy) Prepare(roomID int64, secret string, permanent bool) {
	vrd.Request = "destroy"
	vrd.Room = roomID
	vrd.Secret = secret
	vrd.Permanent = permanent
}

type VideoRoomAllowed struct {
	Request string   `json:"request"`
	Room    int64    `json:"room"`
	Secret  string   `json:"secret,omitempty"`
	Action  string   `json:"action"` // enable,disable,add,remove
	Allowed []string `json:"allowed,omitempty"`
}

type VideoRoomKick struct {
	Request string `json:"request"`
	Room    int64  `json:"room"`
	Secret  string `json:"secret,omitempty"`
	ID      int64  `json:"id"`
}

// VideoRoomCommon 简单的共用请求，如 list, exists, listparticipants, leave
type VideoRoomCommon struct {
	Request string `json:"request"`
	Room    int64  `json:"room,omitempty"`
}

// RoomInfo 会议室列表信息
type RoomInfo struct {
	Room            int64  `json:"room"`
	Description     string `json:"description"`
	PinRequired     bool   `json:"pin_required"`
	MaxPublishers   int    `json:"max_publishers"`
	Bitrate         int    `json:"bitrate"`
	BitrateCap      bool   `json:"bitrate_cap"`
	FirFreq         int    `json:"fir_freq"`
	AudioCodec      string `json:"audiocodec"`
	VideoCodec      string `json:"videocodec"`
	Record          bool   `json:"record"`
	RecordDir       string `json:"record_dir,omitempty"`
	LockRecord      bool   `json:"lock_record"`
	NumParticipants int    `json:"num_participants"`
}

// ParticipantsInfo participants info
type ParticipantsInfo struct {
	ID        int64  `json:"id"`
	Display   string `json:"display"`
	Publisher bool   `json:"publisher"`
	Talking   bool   `json:"talking"`
}

// --------- publish and subscibute

// VideoRoomJoin join as publisher or subscriber
type VideoRoomJoin struct {
	Request       string `json:"request"`
	Ptype         string `json:"ptype"`
	Room          int64  `json:"room"`
	Pin           string `json:"pin,omitempty"`
	ID            int64  `json:"id,omitempty"`
	Display       string `json:"display,omitempty"`
	Token         string `json:"token,omitempty"`
	Feed          int64  `json:"feed,omitempty"` // subscriber
	PrivateID     int64  `json:"private_id,omitempty"`
	ClosePC       *bool  `json:"close_pc,omitempty"`
	Audio         *bool  `json:"audio,omitempty"`
	Video         *bool  `json:"video,omitempty"`
	Data          *bool  `json:"data,omitempty"`
	OfferAudio    *bool  `json:"offer_audio,omitempty"`
	OfferVideo    *bool  `json:"offer_video,omitempty"`
	OfferData     *bool  `json:"offer_data,omitempty"`
	SubStream     int    `json:"substream,omitempty"`      // substream to receive (0-2), in case simulcasting is enabled; optional
	Temporal      int    `json:"temporal,omitempty"`       // temporal layers to receive (0-2), in case simulcasting is enabled; optional
	Fallback      int    `json:"fallback,omitempty"`       // How much time (in us, default 250000) without receiving packets will make us drop to the substream below
	SpatialLayer  int    `json:"spatial_layer,omitempty"`  // spatial layer to receive (0-2), in case VP9-SVC is enabled; optional
	TemporalLayer int    `json:"temporal_layer,omitempty"` // temporal layers to receive (0-2), in case VP9-SVC is enabled; optional
}

// AsPublisher join as publisher
func (vrj *VideoRoomJoin) AsPublisher(roomID int64, display string) {
	vrj.Request = "join"
	vrj.Ptype = "publisher"
	vrj.Room = roomID
	vrj.Display = display
	// vrj.Token = token // invitation token, in case the room has an ACL; optional
	// vrj.ID = id // unique ID to register for the publisher; optional, will be chosen by the plugin if missing
}

// AsSubscriber join as subscriber
func (vrj *VideoRoomJoin) AsSubscriber(roomID, feedID int64) {
	vrj.Request = "join"
	vrj.Ptype = "subscriber"
	vrj.Room = roomID
	vrj.Feed = feedID
	vrj.ClosePC = client.Bool(true)
	vrj.Audio = client.Bool(true)
	vrj.Video = client.Bool(true)
	vrj.Data = client.Bool(true)
	vrj.OfferAudio = client.Bool(true)
	vrj.OfferVideo = client.Bool(true)
	vrj.OfferData = client.Bool(true)
}

// VideoPublisher 视频会议发布者信息
type VideoPublisher struct {
	ID         int64  `json:"id"`
	Display    string `json:"display"`
	AudioCodec string `json:"audio_codec,omitempty"`
	VideoCodec string `json:"video_codec,omitempty"`
	Simulcast  bool   `json:"simulcast,omitempty"`
	Talking    bool   `json:"talking,omitempty"`
}

// VideoRoomPublish publish video
type VideoRoomPublish struct {
	Request            string  `json:"request"`
	Audio              bool    `json:"audio"`
	Video              bool    `json:"video"`
	Data               bool    `json:"data"`
	Update             *bool   `json:"update,omitempty"`
	AudioCodec         string  `json:"audiocodec,omitempty"`
	VideoCodec         string  `json:"videocodec,omitempty"`
	Bitrate            *int    `json:"bitrate,omitempty"`
	Record             *bool   `json:"record,omitempty"`
	FileName           *string `json:"filename,omitempty"`
	Display            *string `json:"display,omitempty"`
	AudioLevelAverage  *int    `json:"audio_level_average,omitempty"`
	AudioActivePackets *int    `json:"audio_active_packets,omitempty"`
}

// SetupInit init publish request
func (vrp *VideoRoomPublish) SetupInit(display string) {
	vrp.Request = "publish"
	vrp.Video = true
	vrp.Audio = true
	vrp.Data = true
	vrp.Display = client.String(display)
}

// AsConfigure 准备配置发布选项
func (vrp *VideoRoomPublish) AsConfigure() {
	vrp.Request = "configure"
	vrp.Video = true
	vrp.Audio = true
	vrp.Data = true
}

// VideoRoomSwitch 切换订阅
type VideoRoomSwitch struct {
	Request string `json:"request"`
	Feed    int64  `json:"feed,omitempty"`
	Audio   *bool  `json:"audio,omitempty"`
	Video   *bool  `json:"video,omitempty"`
	Data    *bool  `json:"data,omitempty"`
}

// SetupInit 初始化
func (vrs *VideoRoomSwitch) SetupInit(feed int64) {
	vrs.Request = "switch"
	vrs.Feed = feed
	vrs.Video = client.Bool(true)
	vrs.Audio = client.Bool(true)
	vrs.Data = client.Bool(true)
}

type VideoRoomModerate struct {
	Request   string `json:"request"`
	Secret    string `json:"secret,omitempty"`
	Room      int64  `json:"room"`
	ID        int64  `json:"id"`
	MuteAudio *bool  `json:"mute_audio,omitempty"`
	MuteVideo *bool  `json:"mute_video,omitempty"`
	MuteData  *bool  `json:"mute_data,omitempty"`
}

func (vrm *VideoRoomModerate) Setup(roomID int64, secret string, pubID int64) {
	vrm.Request = "moderate"
	vrm.Room = roomID
	vrm.Secret = secret
	vrm.ID = pubID
}

// VideoRoomRelayData relay data
type VideoRoomRelayData struct {
	Request   string      `json:"request"`
	RelayData interface{} `json:"relay_data"`
}

// SetPayload set relay data
func (rd *VideoRoomRelayData) SetPayload(v interface{}) {
	rd.Request = "relay_data"
	rd.RelayData = v
}
