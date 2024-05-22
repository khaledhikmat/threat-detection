package agent

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/khaledhikmat/threat-detection-shared/models"
	"github.com/khaledhikmat/threat-detection-shared/service/config"
	"github.com/khaledhikmat/threat-detection-shared/service/publisher"
	"github.com/khaledhikmat/threat-detection-shared/service/soicat"
	"github.com/khaledhikmat/threat-detection-shared/service/storage"
	"github.com/khaledhikmat/threat-detection-shared/utils"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// There is one agent per camera.
func Run(canxCtx context.Context, configsvc config.IService, storagesvc storage.IService, publishersvc publisher.IService, commandsStream chan string, capturer string, camera soicat.Camera) error {

	// Create a cemra folder within the recordings folder if not exist
	err := utils.CreateDirIfNotExist(fmt.Sprintf("%s/%s", configsvc.GetCapturer().RecordingsFolder, camera.Name))
	if err != nil {
		return fmt.Errorf("unable to create a folder: %s/%s - error: %v", configsvc.GetCapturer().RecordingsFolder, camera.Name, err)
	}

	// Create a recording stream
	recordingStream := captureRecordingClip(canxCtx, configsvc, storagesvc, publishersvc)

	if configsvc.GetCapturer().AgentMode == "streaming" {
		return runStreaming(canxCtx, configsvc, recordingStream, commandsStream, capturer, camera)
	}

	return runFiles(canxCtx, configsvc, recordingStream, commandsStream, capturer, camera)
}

func runStreaming(canxCtx context.Context,
	configsvc config.IService,
	recordingStream chan models.RecordingClip,
	commandsStream chan string,
	capturer string,
	camera soicat.Camera) error {
	mode := "streaming"
	// Establishing the camera connection without backchannel if no substream
	if camera.RtspURL == "" {
		return fmt.Errorf("capturer %s - agent %s mode %s: no rtsp url found in config, please provide one", capturer, camera.Name, mode)
	}

	rtspClient := NewRTSPClient(camera.RtspURL)
	defer rtspClient.Close()

	err := rtspClient.Connect(canxCtx)
	if err != nil {
		return fmt.Errorf("capturer %s - agent %s mode %s: error connecting to rtsp stream: %v", capturer, camera.Name, mode, err)
	}

	fmt.Printf("capturer.agent - opened RTSP stream: %s\n", camera.RtspURL)

	// Get the video streams from the RTSP server.
	videoStreams, err := rtspClient.GetVideoStreams()
	if err != nil || len(videoStreams) == 0 {
		return fmt.Errorf("capturer %s - agent %s mode %s: no video stream found, might be the wrong codec (we only support H264 for the moment)", capturer, camera.Name, mode)
	}

	// Get the video stream from the RTSP server.
	videoStream := videoStreams[0]

	// Get some information from the video stream.
	width := videoStream.Width
	height := videoStream.Height

	// Override config values as well
	camera.CaptureWidth = width
	camera.CaptureHeight = height

	// Capture errors
	errorsStream := captureErrors(canxCtx, capturer, camera, mode)

	// Create a packet stream
	packetsStream := make(chan Packet, 10)
	defer close(packetsStream)

	go func() {
		err := rtspClient.Start(canxCtx, errorsStream, packetsStream, camera)
		if err != nil {
			errorsStream <- fmt.Errorf("capturer %s - agent %s mode %s - starting an RSTP client failed: %v", capturer, camera.Name, mode, err)
			return
		}
	}()

	// Capture stream and write mp4 clips to destination (i.e. disk, S3, etc).
	go func() {
		CaptureStream(canxCtx, configsvc, errorsStream, packetsStream, recordingStream, capturer, camera)
	}()

	// Wait for cancellation, command or periodic timer
	for {
		select {
		case <-canxCtx.Done():
			fmt.Printf("capturer %s - agent %s context cancelled...existing!!!\n", capturer, camera.Name)
			return (canxCtx).Err()
		case cmd := <-commandsStream:
			fmt.Printf("capturer %s - agent %s mode %s - command %s\n", capturer, camera.Name, mode, cmd)
			if cmd == "Start" {
				fmt.Printf("capturer %s - agent %s mode %s - start command processor\n", capturer, camera.Name, mode)
			} else if cmd == "Stop" {
				fmt.Printf("capturer %s - agent %s mode %s - stop command processor\n", capturer, camera.Name, mode)
			} else if cmd == "Pause" {
				fmt.Printf("capturer %s - agent %s mode %s - pause command processor\n", capturer, camera.Name, mode)
				err := rtspClient.Pause()
				if err != nil {
					errorsStream <- fmt.Errorf("capturer %s - agent %s mode %s - pausing an RSTP client failed: %v", capturer, camera.Name, mode, err)
				}
			} else if cmd == "Resume" {
				fmt.Printf("capturer %s - agent %s mode %s - resume command processor\n", capturer, camera.Name, mode)
				err := rtspClient.Resume()
				if err != nil {
					errorsStream <- fmt.Errorf("capturer %s - agent %s mode %s - resuming an RSTP client failed: %v", capturer, camera.Name, mode, err)
				}
			}
		case <-time.After(time.Duration(20 * time.Second)):
			fmt.Printf("capturer %s - agent %s mode %s - timeout....perform periodic tasks...\n", capturer, camera.Name, mode)
		}
	}
}

func runFiles(canxCtx context.Context, configsvc config.IService, recordingStream chan models.RecordingClip, commandsStream chan string, capturer string, camera soicat.Camera) error {
	mode := "files"

	// Capture errors
	errorsStream := captureErrors(canxCtx, capturer, camera, mode)

	// Wait for cancellation, command or periodic timer
	for {
		select {
		case <-canxCtx.Done():
			fmt.Printf("capturer %s - agent %s mode %s - context cancelled...existing!!!\n", capturer, camera.Name, mode)
			return (canxCtx).Err()
		case cmd := <-commandsStream:
			fmt.Printf("capturer %s - agent %s mode %s - command %s\n", capturer, camera.Name, mode, cmd)
			if cmd == "Start" {
				fmt.Printf("capturer %s - agent %s mode %s - start command processor\n", capturer, camera.Name, mode)
			} else if cmd == "Stop" {
				fmt.Printf("capturer %s - agent %s mode %s - stop command processor\n", capturer, camera.Name, mode)
			} else if cmd == "Pause" {
				fmt.Printf("capturer %s - agent %s mode %s - pause command processor\n", capturer, camera.Name, mode)
			} else if cmd == "Resume" {
				fmt.Printf("capturer %s - agent %s mode %s - resume command processor\n", capturer, camera.Name, mode)
			}
		case <-time.After(time.Duration(3 * time.Second)):
			fmt.Printf("capturer %s - agent %s mode %s - timeout....perform periodic tasks...\n", capturer, camera.Name, mode)

			err := produceClip(recordingStream, configsvc, configsvc.GetCapturer().SamplesFolder, configsvc.GetCapturer().RecordingsFolder, capturer, camera)
			if err != nil {
				errorsStream <- fmt.Errorf("capturer %s - agent %s mode %s - uploading files: %v", capturer, camera.Name, mode, err)
			}
		}
	}
}

func captureErrors(canxCtx context.Context, capturer string, camera soicat.Camera, mode string) chan interface{} {
	// Create an error stream
	errorsStream := make(chan interface{}, 10)

	// Run an error processor to capture agent errors
	go func() {
		defer close(errorsStream)

		for {
			select {
			case <-canxCtx.Done():
				fmt.Printf("capturer %s - agent %s mode %s - error processor context cancelled\n", capturer, camera.Name, mode)
				return
			case err := <-errorsStream:
				fmt.Printf("capturer %s - agent %s mode %s - error processed received from a downstream error: %v\n", capturer, camera.Name, mode, err)
				// TODO: Agent errors can be sent to key/value storage
				// where the key = capturer_camera_name_ts
				// and the value = error.Error()
			}
		}
	}()

	return errorsStream
}

func captureRecordingClip(canxCtx context.Context,
	configsvc config.IService,
	storagesvc storage.IService,
	publishersvc publisher.IService) chan models.RecordingClip {
	// Create a recording stream
	recordingStream := make(chan models.RecordingClip, 10)

	go func() {
		defer close(recordingStream)

		// Recording processor
		for {
			select {
			case <-canxCtx.Done():
				fmt.Println("recording processor context cancelled...")
				return
			case recording := <-recordingStream:
				fmt.Printf("recording processor file %s received\n", recording.LocalReference)

				// Upload to Cloud Storage i.e. S3, Azure Storage, etc
				url, err := storagesvc.StoreRecordingClip(canxCtx, recording)
				if err != nil {
					fmt.Printf("unable to store recording clip: %s in %s due to: %v\n", recording.LocalReference, configsvc.GetRuntime(), err)
				}
				recording.CloudReference = url
				recording.StorageProvider = configsvc.GetRuntime()
				fmt.Printf("Uploaded %s to %s => %s\n", recording.LocalReference, configsvc.GetRuntime(), recording.CloudReference)

				// Publish event
				fmt.Printf("Publishing %s recording clip\n", recording.CloudReference)
				err = publishersvc.PublishRecordingClip(canxCtx, models.ThreatDetectionPubSub, models.RecordingsTopic, recording)
				if err != nil {
					fmt.Printf("unable to publish event: %s %v\n", recording.LocalReference, err)
				}

				// Delete local file
				fmt.Printf("Deleting %s from local\n", recording.LocalReference)
				err = os.Remove(recording.LocalReference)
				if err != nil {
					fmt.Printf("unable to remove file: %s %v\n", recording.LocalReference, err)
				}
			}
		}
	}()

	return recordingStream
}

func produceClip(recordingStream chan models.RecordingClip,
	configsvc config.IService,
	samplesFolder, recordingsFolder string,
	capturer string,
	camera soicat.Camera) error {
	files, err := os.Open(samplesFolder)
	if err != nil {
		return fmt.Errorf("unable to open samples foler %s: %v", samplesFolder, err)
	}
	defer files.Close()

	ffs, err := files.Readdir(-1) // read all the files
	if err != nil {
		return fmt.Errorf("unable to read sample files: %v", err)
	}

	// Pick random one
	file := ffs[rand.Intn(len(ffs))]

	source, err := os.Open(fmt.Sprintf("%s/%s", samplesFolder, file.Name()))
	if err != nil {
		return fmt.Errorf("unable to open source file: %s %v", file.Name(), err)
	}
	defer source.Close()

	destination, err := os.Create(fmt.Sprintf("%s/%s/%s_%s_%s", recordingsFolder, camera.Name, capturer, uuid.New().String(), file.Name()))
	if err != nil {
		return fmt.Errorf("unable to create dest file: %s %v", file.Name(), err)
	}

	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("unable to copy source %s to dest file %s: %v", source.Name(), destination.Name(), err)
	}

	// Send the recording clip via the storage stream
	recordingStream <- models.RecordingClip{
		ID:                uuid.NewString(),
		LocalReference:    destination.Name(),
		CloudReference:    "",
		StorageProvider:   configsvc.GetRuntime(),
		Capturer:          capturer,
		Camera:            camera.Name,
		Region:            camera.Region,
		Location:          camera.Location,
		Priority:          camera.Priority,
		Analytics:         camera.Analytics,
		AlertTypes:        camera.AlertTypes,
		MediaIndexerTypes: camera.MediaIndexerTypes,
		Frames:            122,
		BeginTime:         time.Now().Add(-3 * time.Second).Format(models.Layout),
		EndTime:           time.Now().Format(models.Layout),
	}

	return nil
}
