package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/finove/webrtctest/client"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

func main() {
	var isSend, isCli bool
	var feedID int64
	var cli uClient
	var err error
	flag.BoolVar(&isSend, "send", false, "send mode")
	flag.BoolVar(&isCli, "cli", false, "cli test mode")
	flag.Int64Var(&feedID, "feed", 0, "video room feed id")
	flag.Parse()
	if isCli || feedID > 0 {
		err = cli.Init(janusAddress, janusSecret)
		if err != nil {
			panic(err)
		}
		if feedID == 0 {
			cli.listparticipants(1234)
			return
		}
	}
	log.Printf("is send %v", isSend)
	config := webrtc.Configuration{
		// ICEServers: []webrtc.ICEServer{
		// 	{
		// 		URLs: []string{"stun:stun.1.google.com:19302"},
		// 	},
		// },
		SDPSemantics: webrtc.SDPSemanticsPlanB,
		// SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
		RTCPMuxPolicy: webrtc.RTCPMuxPolicyRequire,
		BundlePolicy:  webrtc.BundlePolicyMaxBundle,
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	peerConnection.CreateDataChannel("application", &webrtc.DataChannelInit{
		Negotiated: client.Bool(false),
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		codec := track.Codec()
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if errSend != nil {
					fmt.Println(errSend)
				}
			}
		}()
		log.Printf("got codec %s", codec.MimeType)
		SaveRemoteTrack("out3", track)
	})
	iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			iceConnectedCtxCancel()
		}
	})

	if isSend && feedID == 1 {
		// Create a audio track
		audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion1")
		if err != nil {
			panic(err)
		}
		_, err = peerConnection.AddTrack(audioTrack)
		if err != nil {
			panic(err)
		}

		vp8Track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "pion2")
		if err != nil {
			panic(err)
		} else if _, err = peerConnection.AddTrack(vp8Track); err != nil {
			panic(err)
		}

		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}
		// log.Printf("local %s sdp %s", offer.Type.String(), offer.SDP)
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
		peerConnection.SetLocalDescription(offer)
		<-gatherComplete
		// wait anser
		answer := webrtc.SessionDescription{}
		// Decode(MustReadStdin(), &answer)
		if err = cli.JoinRoom(1234); err != nil {
			panic(err)
		}
		if answer.SDP, err = cli.Publish(peerConnection.LocalDescription().SDP); err != nil {
			panic(err)
		}
		answer.Type = webrtc.SDPTypeAnswer
		peerConnection.SetRemoteDescription(answer)

		<-iceConnectedCtx.Done()
		go SendVP8Video(context.Background(), "out3.ivf", vp8Track)
		go SendOggAudio(context.Background(), "out3.opus", audioTrack)

	} else {

		offer := webrtc.SessionDescription{}
		if feedID > 0 {
			if _, offer.SDP, err = cli.Subscrite(1234, feedID); err != nil {
				panic(err)
			}
			offer.Type = webrtc.SDPTypeOffer
		} else {
			Decode(MustReadStdin(), &offer)
		}
		log.Printf("offer %s", offer.SDP)

		err = peerConnection.SetRemoteDescription(offer)
		if err != nil {
			log.Printf("offer %s fail:%v", offer.SDP, err)
			// panic(err)
		}

		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			log.Printf("offer %s fail:%v", offer.SDP, err)
			// panic(err)
		}
		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			panic(err)
		}
		<-gatherComplete
		if feedID > 0 {
			answerLoc := peerConnection.LocalDescription()
			err = cli.SubscriteStart(answerLoc.SDP)
			if err != nil {
				panic(err)
			}
		} else {
			log.Println(Encode(*peerConnection.LocalDescription()))
		}
	}

	select {}
}
