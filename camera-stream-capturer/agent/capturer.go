package agent

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/yapingcat/gomedia/go-mp4"

	"github.com/khaledhikmat/threat-detection-shared/equates"
	"github.com/khaledhikmat/threat-detection-shared/service/config"
	"github.com/khaledhikmat/threat-detection-shared/service/soicat"
)

func CaptureStream(canxCtx context.Context,
	configsvc config.IService,
	errorsStream chan interface{},
	packetsStream chan Packet,
	storageStream chan equates.RecordingClip,
	capturer string,
	camera soicat.Camera) {
	var file *os.File
	var myMuxer *mp4.Movmuxer
	var videoTrack uint32

	// Get as many packets we need.
	recordingStatus := "idle"
	recordingStart := time.Now().Unix()
	recordingStartTS := time.Now()
	frames := 0

	for {
		select {
		case <-canxCtx.Done():
			fmt.Printf("CaptureStream context is cancelled\n")
			return
		case pkt := <-packetsStream:
			if recordingStatus == "idle" {
				// Start recording only when we receive a key frame packet
				if pkt.IsVideo && pkt.IsKeyFrame {
					recordingStart = time.Now().Unix()
					recordingStartTS = time.Now()
					fullName := fmt.Sprintf("%s/%s/%s.mp4", configsvc.GetCapturer().RecordingsFolder, camera.Name, strconv.FormatInt(recordingStart, 10))

					var err error
					file, err = os.Create(fullName)
					if err != nil {
						errorsStream <- fmt.Errorf("capturestream: %v", err.Error())
					}

					myMuxer, _ = mp4.CreateMp4Muxer(file)
					// We choose between H264 and H265
					width := camera.CaptureWidth
					height := camera.CaptureHeight
					widthOption := mp4.WithVideoWidth(uint32(width))
					heightOption := mp4.WithVideoHeight(uint32(height))

					// Write video header
					if pkt.Codec == "H264" {
						videoTrack = myMuxer.AddVideoTrack(mp4.MP4_CODEC_H264, widthOption, heightOption)
					} else if pkt.Codec == "H265" {
						videoTrack = myMuxer.AddVideoTrack(mp4.MP4_CODEC_H265, widthOption, heightOption)
					}

					// Write video packet
					ttime := uint64(pkt.Time.Milliseconds())
					if err := myMuxer.Write(videoTrack, pkt.Data, ttime, ttime); err != nil {
						errorsStream <- fmt.Errorf("capturestream: %v", err.Error())
					}

					// Reset the frames counter (header + 1st packet)
					frames = 2

					// Switch to recording mode
					recordingStatus = "recording"
				}
			} else {
				// Stop recording only if we have exceeded the timeout and a keyframe arrives
				if pkt.IsVideo && pkt.IsKeyFrame && ((recordingStart + camera.MaxLengthRecording) <= time.Now().Unix()) {

					// Write video trailer
					if err := myMuxer.WriteTrailer(); err != nil {
						errorsStream <- fmt.Errorf("capturestream: %v", err.Error())
					}

					// Include the trailer
					frames++

					fmt.Printf("CaptureStream - file save: %s - frames: %d\n", file.Name(), frames)

					// Close the file and cleanup muxer
					file.Close()

					// Send the recording clip via the storage stream
					storageStream <- equates.RecordingClip{
						ID:                uuid.NewString(),
						LocalReference:    fmt.Sprintf("%s/%s/%s", configsvc.GetCapturer().RecordingsFolder, camera.Name, file.Name()),
						CloudReference:    "",
						Capturer:          capturer,
						Camera:            camera.Name,
						Region:            camera.Region,
						Location:          camera.Location,
						Priority:          camera.Priority,
						Analytics:         camera.Analytics,
						AlertTypes:        camera.AlertTypes,
						MediaIndexerTypes: camera.MediaIndexerTypes,
						Frames:            frames,
						BeginTime:         recordingStartTS.Format(equates.Layout),
						EndTime:           time.Now().Format(equates.Layout),
					}

					// Remove the file reference
					file = nil

					// Switch to idle mode
					recordingStatus = "idle"
				} else if pkt.IsVideo {
					// Write video packet
					ttime := uint64(pkt.Time.Milliseconds())
					if err := myMuxer.Write(videoTrack, pkt.Data, ttime, ttime); err != nil {
						errorsStream <- fmt.Errorf("capturestream: %v", err.Error())
					}

					// Add a video frame
					frames++
				}
			}
		}
	}
}
