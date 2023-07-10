package main

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/webrtc/v3"
)

var config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
		{
			URLs:       []string{"turn:34.210.41.65:3478"},
			Username:   "blakkhawkcaller",
			Credential: "L8a3MmfdgVSFqZam",
		},
	},
}

func NewWebRTCSession(request *InitiateWebRTCSessionRequest, mqttClient mqtt.Client) *WebRTCSession {
	var sessionPeerConnection, err = webrtc.NewPeerConnection(config)

	if err != nil {
		// FIXME handle exception!!
		log.Printf("Failed creating a peer connection with error: %s", err)
	}

	return &WebRTCSession{request.SessionId, request.CallerId, sessionPeerConnection, mqttClient}
}

type WebRTCSession struct {
	Id             string
	CallerId       string
	PeerConnection *webrtc.PeerConnection
	MQTTClient     mqtt.Client
}

// Signalling Message Handler
func (this *WebRTCSession) handleSignallingMessage(client mqtt.Client, message mqtt.Message) {

	messagePayload := message.Payload()
	log.Printf("handleSignallingMessage received message: %s from topic: %s\n", message.Payload(), message.Topic())

	brokerMessage := BrokerMessage{}
	json.Unmarshal([]byte(messagePayload), &brokerMessage)
	log.Printf("brokerMessage: %#v", brokerMessage)

	if brokerMessage.Type == "offer" {
		offerMessage := SDPMessage{}
		json.Unmarshal([]byte(messagePayload), &offerMessage)
		log.Printf("Message SDP: %#v\n", offerMessage.SDP)

		SetupLocalOfferAndAnswer(this, offerMessage)

	} else if brokerMessage.Type == "new-icecandidate" {
		newIceCandidate := ICECandidateMessage{}
		json.Unmarshal([]byte(messagePayload), &newIceCandidate)
		log.Printf("Tring to AddICECandidate: %#v\n", newIceCandidate)

		// Add a new ICECandidate
		if err := this.PeerConnection.AddICECandidate(newIceCandidate.ICECandidate); err != nil {
			log.Printf("AddICECandidate: %s\n", err)
		}
	}
}

// send message to client
func (this *WebRTCSession) signalClient(serializedMessage []byte) {
	this.MQTTClient.Publish(this.CallerId, 2, false, serializedMessage)
}

// Message structures
type BrokerMessage struct {
	Type string `json:"type"`
}

type SDPMessage struct {
	Type   string                    `json:"type"`
	ToId   string                    `json:"toId"`
	FromId string                    `json:"fromId"`
	SDP    webrtc.SessionDescription `json:"sdp"`
}

type ICECandidateMessage struct {
	Type         string                  `json:"type"`
	ToId         string                  `json:"toId"`
	FromId       string                  `json:"fromId"`
	ICECandidate webrtc.ICECandidateInit `json:"icecandidate"`
}

// private
func SetupLocalOfferAndAnswer(
	webrtcSession *WebRTCSession,
	offerMessage SDPMessage) {

	peerConnection := webrtcSession.PeerConnection
	if rtpsender, err := peerConnection.AddTrack(GetOutboundVideoTrack()); err != nil || rtpsender != nil {
		if err != nil {
			log.Printf("AddTrack Error %v\n", err)
		}
		if rtpsender != nil {
			log.Printf("AddTrack rtpsender %#v\n", rtpsender)
		}
	}

	peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {

		if candidate != nil {
			// Send out the ICECandidate on signaling
			newIceCandidateMessage, _ := json.Marshal(ICECandidateMessage{
				Type:         "new-icecandidate",
				ToId:         offerMessage.FromId,
				FromId:       offerMessage.ToId,
				ICECandidate: candidate.ToJSON(),
			})
			webrtcSession.signalClient(newIceCandidateMessage)
		}

	})

	// Set the remote SessionDescription
	if err := peerConnection.SetRemoteDescription(offerMessage.SDP); err != nil {
		log.Printf("SetRemoteDescription: %v", err)
	}

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Printf("CreateAnswer: %v", err)
	} else if err = peerConnection.SetLocalDescription(answer); err != nil {
		log.Printf("SetLocalDescription: %v", err)
	}

	// Respond the answers
	registerMessage, _ := json.Marshal(SDPMessage{
		Type:   "answer",
		ToId:   offerMessage.FromId,
		FromId: offerMessage.ToId,
		SDP:    answer,
	})
	webrtcSession.signalClient(registerMessage)
}
