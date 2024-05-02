package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/khaledhikmat/threat-detection/shared/service/soicat"
)

// There is one agent per camera. It is responsible for:
// 1. Processing agent errors
// 2. Starting RTSP client
// 3. Capturing camera stream packets which produces MP4 video clips
// 4. Transferring the video clips to a Cloud storage
// 5. Listen to commands from the capturer to start, stop, pause and resume
func Run(canxCtx context.Context, commandsStream chan string, capturer string, camera soicat.Camera) error {
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

	// Run an error processor to capture agent errors
	go func() {
		defer close(errorsStream)

		for {
			select {
			case <-canxCtx.Done():
				fmt.Printf("capturer %s - agent %s error processor context cancelled\n", capturer, camera.Name)
				return
			case err := <-errorsStream:
				fmt.Printf("capturer %s - agent %s error processed received from a downstream error: %v\n", capturer, camera.Name, err)
				// TODO: Agent errors can be sent to key/value storage
				// where the key = capturer_camera_name_ts
				// and the value = error.Error()
			}
		}
	}()

	// Create a packet stream
	packetsStream := make(chan Packet, 10)
	defer close(packetsStream)

	go func() {
		err := rtspClient.Start(canxCtx, errorsStream, packetsStream, camera)
		if err != nil {
			errorsStream <- fmt.Errorf("Agent %s - starting an RSTP client failed: %v", camera.Name, err)
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
			fmt.Printf("capturer %s - agent %s context cancelled...existing!!!\n", capturer, camera.Name)
			return (canxCtx).Err()
		case cmd := <-commandsStream:
			fmt.Printf("capturer %s - agent %s - command %s\n", capturer, camera.Name, cmd)
			if cmd == "Pause" {
				err := rtspClient.Pause()
				if err != nil {
					errorsStream <- fmt.Errorf("Agent %s - pausing an RSTP client failed: %v", camera.Name, err)
				}
			} else if cmd == "Resume" {
				err := rtspClient.Resume()
				if err != nil {
					errorsStream <- fmt.Errorf("Agent %s - resuming an RSTP client failed: %v", camera.Name, err)
				}
			}
		case <-time.After(time.Duration(100 * time.Second)):
			fmt.Printf("capturer %s - agent %s timeout....do something periodic here!!!\n", capturer, camera.Name)
		}
	}
}
