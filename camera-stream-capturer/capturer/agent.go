package capturer

import (
	"context"
	"fmt"
	"time"

	"github.com/khaledhikmat/threat-detection/shared/service/config"
	"github.com/khaledhikmat/threat-detection/shared/service/soicat"
)

func Run(canxCtx context.Context, cfgsvc config.IService, camera soicat.Camera) error {
	// Currently only support H264 encoded cameras, this will change.
	// Establishing the camera connection without backchannel if no substream
	if camera.RtspURL == "" {
		return fmt.Errorf("no rtsp url found in config, please provide one")
	}

	rtspClient := NewRTSPClient(camera.RtspURL)
	defer rtspClient.Close()

	err := rtspClient.Connect(canxCtx)
	if err != nil {
		return fmt.Errorf("error connecting to rtsp stream: %v", err)
	}

	fmt.Printf("capturer.agent - opened RTSP stream: %s\n", camera.RtspURL)

	// Get the video streams from the RTSP server.
	videoStreams, err := rtspClient.GetVideoStreams()
	if err != nil || len(videoStreams) == 0 {
		return fmt.Errorf("capturer.agent: no video stream found, might be the wrong codec (we only support H264 for the moment)")
	}

	// Get the video stream from the RTSP server.
	videoStream := videoStreams[0]

	// Get some information from the video stream.
	width := videoStream.Width
	height := videoStream.Height

	// Override config values as well
	camera.CaptureWidth = width
	camera.CaptureHeight = height

	// Create an error stream
	errorsStream := make(chan interface{}, 10)
	defer close(errorsStream)

	// Run an error processor to capture agent errors
	go func() {
		for {
			select {
			case <-canxCtx.Done():
				fmt.Printf("agent %s error processor context cancelled\n", camera.Name)
				return
			case err := <-errorsStream:
				fmt.Printf("agent %s error processed received from a downstream error: %v\n", camera.Name, err)
			}
		}
	}()

	// Create a packet stream
	packetsStream := make(chan Packet, 10)
	defer close(packetsStream)

	go func() {
		err := rtspClient.Start(canxCtx, errorsStream, packetsStream, camera)
		if err != nil {
			errorsStream <- fmt.Errorf("capturer.agent - starting an RSTP client failed: %v", err)
			return
		}
	}()

	// Capture stream and write mp4 clips to destination (i.e. disk, S3, etc).
	go func() {
		CaptureStream(canxCtx, errorsStream, packetsStream, camera)
	}()

	// Wait
	for {
		select {
		case <-canxCtx.Done():
			fmt.Println("Agent context cancelled...existing!!!")
			return (canxCtx).Err()
		case <-time.After(time.Duration(100 * time.Second)):
			fmt.Println("Timeout....do something periodic here!!!")
		}
	}
}
