package main

import (
	"os"
	"os/signal"

	argparser "github.com/alexflint/go-arg"
)

/* Arguments structure for EdgeResponder */
var args struct {
	RTSPFeedURL  string `arg:"-f,--rtspfeed,required" help:"RTSP Feed URL to stream on WebRTC connection"`
	MQTTEndpoint string `arg:"-s,--mqttendpoint,required" help:"MQTT connection endpoint with port."`
	DeviceId     string `arg:"-i,--deviceid,required" help:"UniqueId registered in blakkhawk backend"`
}

func main() {

	argparser.MustParse(&args)

	videoTrackReady := make(chan interface{})
	go StartRTSPStream(args.RTSPFeedURL, videoTrackReady)
	<-videoTrackReady

	osinterrupt := make(chan os.Signal, 1)
	signal.Notify(osinterrupt, os.Interrupt)

	ConnectAndRegister(args.MQTTEndpoint, args.DeviceId)

	<-osinterrupt
	// Perform cleanup
}
