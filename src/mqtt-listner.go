package main

import (
	"encoding/json"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Global variables
var (
	mqttClient mqtt.Client
	clientId   string
)

func ConnectAndRegister(mqttEndpoint string, deviceId string) {

	clientOpts := mqtt.NewClientOptions()

	clientOpts.AddBroker(mqttEndpoint)
	clientOpts.SetClientID(deviceId)
	clientOpts.SetDefaultPublishHandler(messageHandler)
	clientOpts.SetOnConnectHandler(onConnectHandler)

	log.Printf("Connecting to %s with clientId: %s", mqttEndpoint, deviceId)
	clientId = deviceId
	mqttClient = mqtt.NewClient(clientOpts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		// TODO: Instead surface this to a global channel and restart the connection process.
		panic(token.Error())
	}
}

/* Connection update Handlers */
var onConnectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Printf("Successfully connected to MQTT Broker and subscribed to: %s", "initiatesession-"+clientId)
	client.Subscribe("initiatesession-"+clientId, 2, onInitiateSessionRequestHandler)
}

/* MQTT Message Handlers */
var messageHandler mqtt.MessageHandler = func(client mqtt.Client, message mqtt.Message) {
	log.Printf("DefaultMessageHandler received message: %s from topic: %s\n", message.Payload(), message.Topic())
}

var onInitiateSessionRequestHandler mqtt.MessageHandler = func(client mqtt.Client, message mqtt.Message) {
	// Create a unique WebRTC call handler
	log.Printf("onInitiateSessionRequestHandler received message: %s from topic: %s serialized: %#v\n",
		message.Payload(),
		message.Topic(),
		parseInitiateWebRTCSessionRequest(message.Payload()))

	webrtcSession := NewWebRTCSession(parseInitiateWebRTCSessionRequest(message.Payload()), client)
	client.Subscribe(webrtcSession.Id, 2, webrtcSession.handleSignallingMessage)
}

/* Server Messages Structures */
type InitiateWebRTCSessionRequest struct {
	SessionId string `json:"sessionId"`
	CallerId  string `json:"callerId"`
}

func parseInitiateWebRTCSessionRequest(serializedBytes []byte) *InitiateWebRTCSessionRequest {
	messageObj := new(InitiateWebRTCSessionRequest)
	json.Unmarshal(serializedBytes, messageObj)
	return messageObj
}
