package main

import (
	"io"
	"log"
	"time"

	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/codec/h264parser"
	"github.com/deepch/vdk/format/rtsp"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

var outboundVideoTrack *webrtc.TrackLocalStaticSample

func GetOutboundVideoTrack() *webrtc.TrackLocalStaticSample {
	return outboundVideoTrack
}

func StartRTSPStream(rtspURL string, videoTrackReady chan<- interface{}) {

	outboundVideoTrack, _ = webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType: "video/h264",
		}, "cam-rtsp", uuid.New().String())
	close(videoTrackReady)

	// const rtspURL = "rtsp://fD0xzWUv:AMo1a5eZRvAqXFXq@192.168.0.247:554/live/ch1"
	annexbNALUStartCode := func() []byte { return []byte{0x00, 0x00, 0x00, 0x01} }
	for {
		session, err := rtsp.Dial(rtspURL)
		if err != nil {
			panic(err)
		}
		session.RtpKeepAliveTimeout = 10 * time.Second

		codecs, err := session.Streams()
		if err != nil {
			panic(err)
		}
		for i, t := range codecs {
			log.Println("Stream", i, "is of type", t.Type().String())
		}
		if codecs[0].Type() != av.H264 {
			panic("RTSP feed must begin with a H264 codec")
		}
		if len(codecs) != 1 {
			log.Println("Ignoring all but the first stream.")
		}

		var previousTime time.Duration
		for {
			pkt, err := session.ReadPacket()
			if err != nil {
				break
			}

			if pkt.Idx != 0 {
				//audio or other stream, skip it
				continue
			}

			pkt.Data = pkt.Data[4:]

			// For every key-frame pre-pend the SPS and PPS
			if pkt.IsKeyFrame {
				pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
				pkt.Data = append(codecs[0].(h264parser.CodecData).PPS(), pkt.Data...)
				pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
				pkt.Data = append(codecs[0].(h264parser.CodecData).SPS(), pkt.Data...)
				pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
			}

			bufferDuration := pkt.Time - previousTime
			previousTime = pkt.Time

			if err = outboundVideoTrack.WriteSample(media.Sample{Data: pkt.Data, Duration: bufferDuration}); err != nil && err != io.ErrClosedPipe {
				log.Printf("outboundVideoTrack.WriteSample error: %s", err)
				panic(err)
			}
		}

		if err = session.Close(); err != nil {
			log.Println("session close error", err)
		}

		time.Sleep(5 * time.Second)
	}
}
