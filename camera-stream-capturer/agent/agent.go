package agent

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/kerberos-io/agent/machinery/src/capture"
	"github.com/kerberos-io/agent/machinery/src/log"
	"github.com/kerberos-io/agent/machinery/src/models"
	"github.com/kerberos-io/agent/machinery/src/packets"
)

func Run(canxCtx context.Context, canxFn context.CancelFunc, configDir string, configuration *models.Configuration) error {
	// We create a capture object, this will contain all the streaming clients.
	// And allow us to extract media from within difference places in the agent.
	capture := capture.Capture{
		RTSPClient:    nil,
		RTSPSubClient: nil,
	}

	// Bootstrapping the agent
	communication := models.Communication{
		Context:         &canxCtx,
		CancelContext:   &canxFn,
		HandleBootstrap: make(chan string, 1),
	}

	// Initiate the packet counter, this is being used to detect
	// if a camera is going blocky, or got disconnected.
	var packageCounter atomic.Value
	packageCounter.Store(int64(0))
	communication.PackageCounter = &packageCounter

	var packageCounterSub atomic.Value
	packageCounterSub.Store(int64(0))
	communication.PackageCounterSub = &packageCounterSub

	// This is used when the last packet was received (timestamp),
	// this metric is used to determine if the camera is still online/connected.
	var lastPacketTimer atomic.Value
	packageCounter.Store(int64(0))
	communication.LastPacketTimer = &lastPacketTimer

	var lastPacketTimerSub atomic.Value
	packageCounterSub.Store(int64(0))
	communication.LastPacketTimerSub = &lastPacketTimerSub

	// This is used to understand if we have a working Kerberos Hub connection
	// cloudTimestamp will be updated when successfully sending heartbeats.
	var cloudTimestamp atomic.Value
	cloudTimestamp.Store(int64(0))
	communication.CloudTimestamp = &cloudTimestamp

	return bootstrap(configDir, configuration, &communication, &capture)
}

func bootstrap(configDir string, configuration *models.Configuration, communication *models.Communication, captureDevice *capture.Capture) error {
	fmt.Println("agent.bootstrap: creating processing threads.")
	config := configuration.Config

	// Currently only support H264 encoded cameras, this will change.
	// Establishing the camera connection without backchannel if no substream
	rtspURL := config.Capture.IPCamera.RTSP
	if rtspURL == "" {
		return fmt.Errorf("agent.bootstrap: no rtsp url found in config, please provide one.")
	}

	rtspClient := captureDevice.SetMainClient(rtspURL)
	defer rtspClient.Close()

	err := rtspClient.Connect(*communication.Context)
	if err != nil {
		return fmt.Errorf("components.Kerberos.RunAgent(): error connecting to RTSP stream: %v", err)
	}

	fmt.Println("components.Kerberos.RunAgent(): opened RTSP stream: " + rtspURL)

	// Get the video streams from the RTSP server.
	videoStreams, err := rtspClient.GetVideoStreams()
	if err != nil || len(videoStreams) == 0 {
		return fmt.Errorf("components.Kerberos.RunAgent(): no video stream found, might be the wrong codec (we only support H264 for the moment)")
	}

	// Get the video stream from the RTSP server.
	videoStream := videoStreams[0]

	// Get some information from the video stream.
	width := videoStream.Width
	height := videoStream.Height

	// Set config values as well
	configuration.Config.Capture.IPCamera.Width = width
	configuration.Config.Capture.IPCamera.Height = height

	var queue *packets.Queue

	// Create a packet queue, which is filled by the HandleStream routing
	// and consumed by all other routines: motion, livestream, etc.
	if config.Capture.PreRecording <= 0 {
		config.Capture.PreRecording = 1
		fmt.Printf("components.Kerberos.RunAgent(): Prerecording value not found in config or invalid value! Found %d\n" + strconv.FormatInt(config.Capture.PreRecording, 10))
	}

	// We are creating a queue to store the RTSP frames in, these frames will be
	// processed by the different consumers: motion detection, recording, etc.
	queue = packets.NewQueue()
	communication.Queue = queue
	defer communication.Queue.Close()

	// Set the maximum GOP count, this is used to determine the pre-recording time.
	log.Log.Info("components.Kerberos.RunAgent(): SetMaxGopCount was set with: " + strconv.Itoa(int(config.Capture.PreRecording)+1))
	queue.SetMaxGopCount(int(config.Capture.PreRecording) + 1) // GOP time frame is set to prerecording (we'll add 2 gops to leave some room).
	err = queue.WriteHeader(videoStreams)
	if err != nil {
		return err
	}

	go rtspClient.Start(*communication.Context, "main", queue, configuration, communication)

	// Main stream is connected and ready to go.
	communication.MainStreamConnected = true

	// Handle recording, will write an mp4 to disk.
	go capture.HandleRecordStream(queue, configDir, configuration, communication, rtspClient)

	// Handle Upload to cloud provider (Kerberos Hub, Kerberos Vault and others)
	//go cloud.HandleUpload(configDir, configuration, communication)

	// If we reach this point, we have a working RTSP connection.
	communication.CameraConnected = true

	// Wait
	for {
		select {
		case <-(*communication.Context).Done():
			fmt.Println("Agent context cancelled...existing!!!")
			return (*communication.Context).Err()
		case <-time.After(time.Duration(100 * time.Second)):
			fmt.Println("Timeout....do something periodic here!!!")
		}
	}
}
