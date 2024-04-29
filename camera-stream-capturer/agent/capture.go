package agent

import (
	"os"
	"strconv"
	"time"

	"github.com/kerberos-io/agent/machinery/src/conditions"
	"github.com/kerberos-io/agent/machinery/src/log"
	"github.com/kerberos-io/agent/machinery/src/models"
	"github.com/kerberos-io/agent/machinery/src/packets"
	"github.com/kerberos-io/agent/machinery/src/utils"
	"github.com/yapingcat/gomedia/go-mp4"
)

func CaptureStream(queue *packets.Queue, configDirectory string, configuration *models.Configuration) {

	config := configuration.Config
	loc, _ := time.LoadLocation(config.Timezone)

	maxRecordingPeriod := config.Capture.MaxLengthRecording // maximum number of seconds to record.

	// Synchronise the last synced time
	now := time.Now().Unix()
	startRecording := now

	if config.FriendlyName != "" {
		config.Name = config.FriendlyName
	}

	// For continuous and motion based recording we will use a single file.
	var file *os.File

	//var cws *cacheWriterSeeker
	var myMuxer *mp4.Movmuxer
	var videoTrack uint32
	var name string

	now = time.Now().Unix()
	timestamp := now
	start := false

	// If continuous record the full length
	recordingPeriod := maxRecordingPeriod
	// Recording file name
	fullName := ""

	// Get as many packets we need.
	var cursorError error
	var pkt packets.Packet
	var nextPkt packets.Packet
	recordingStatus := "idle"
	recordingCursor := queue.Oldest()

	pkt, cursorError = recordingCursor.ReadPacket()

	for cursorError == nil {

		nextPkt, cursorError = recordingCursor.ReadPacket()

		now := time.Now().Unix()

		if start && // If already recording and current frame is a keyframe and we should stop recording
			nextPkt.IsKeyFrame && (timestamp+recordingPeriod-now <= 0 || now-startRecording >= maxRecordingPeriod) {

			// Write the last packet
			ttime := uint64(pkt.Time.Milliseconds())
			if pkt.IsVideo {
				if err := myMuxer.Write(videoTrack, pkt.Data, ttime, ttime); err != nil {
					log.Log.Error("capture.main.HandleRecordStream(continuous): " + err.Error())
				}
			}

			// This will write the trailer a well.
			if err := myMuxer.WriteTrailer(); err != nil {
				log.Log.Error("capture.main.HandleRecordStream(continuous): " + err.Error())
			}

			log.Log.Info("capture.main.HandleRecordStream(continuous): recording finished: file save: " + name)

			// Cleanup muxer
			start = false
			file.Close()
			file = nil

			// Check if need to convert to fragmented using bento
			if config.Capture.Fragmented == "true" && config.Capture.FragmentedDuration > 0 {
				utils.CreateFragmentedMP4(fullName, config.Capture.FragmentedDuration)
			}

			recordingStatus = "idle"
		}

		// If not yet started and a keyframe, let's make a recording
		if !start && pkt.IsKeyFrame {

			// We might have different conditions enabled such as time window or uri response.
			// We'll validate those conditions and if not valid we'll not do anything.
			valid, err := conditions.Validate(loc, configuration)
			if !valid && err != nil {
				log.Log.Debug("capture.main.HandleRecordStream(continuous): " + err.Error() + ".")
				time.Sleep(5 * time.Second)
				continue
			}

			start = true
			timestamp = now

			startRecording = time.Now().Unix() // we mark the current time when the record started.ss
			s := strconv.FormatInt(startRecording, 10) + "_" +
				"6" + "-" +
				"967003" + "_" +
				config.Name + "_" +
				"200-200-400-400" + "_0_" +
				"769"

			name = s + ".mp4"
			fullName = configDirectory + "/data/recordings/" + name

			// Running...
			log.Log.Info("capture.main.HandleRecordStream(continuous): recording started")

			file, err = os.Create(fullName)
			if err != nil {
				log.Log.Error("capture.main.HandleRecordStream(continuous): " + err.Error())
			}

			//cws = newCacheWriterSeeker(4096)
			myMuxer, _ = mp4.CreateMp4Muxer(file)
			// We choose between H264 and H265
			width := configuration.Config.Capture.IPCamera.Width
			height := configuration.Config.Capture.IPCamera.Height
			widthOption := mp4.WithVideoWidth(uint32(width))
			heightOption := mp4.WithVideoHeight(uint32(height))
			if pkt.Codec == "H264" {
				videoTrack = myMuxer.AddVideoTrack(mp4.MP4_CODEC_H264, widthOption, heightOption)
			} else if pkt.Codec == "H265" {
				videoTrack = myMuxer.AddVideoTrack(mp4.MP4_CODEC_H265, widthOption, heightOption)
			}

			ttime := uint64(pkt.Time.Milliseconds())
			if pkt.IsVideo {
				if err := myMuxer.Write(videoTrack, pkt.Data, ttime, ttime); err != nil {
					log.Log.Error("capture.main.HandleRecordStream(continuous): " + err.Error())
				}
			}

			recordingStatus = "started"

		} else if start {
			ttime := uint64(pkt.Time.Milliseconds())
			if pkt.IsVideo {
				if err := myMuxer.Write(videoTrack, pkt.Data, ttime, ttime); err != nil {
					log.Log.Error("capture.main.HandleRecordStream(continuous): " + err.Error())
				}
			}
		}

		pkt = nextPkt
	}

	// We might have interrupted the recording while restarting the agent.
	// If this happens we need to check to properly close the recording.
	if cursorError != nil {
		if recordingStatus == "started" {
			// This will write the trailer a well.
			if err := myMuxer.WriteTrailer(); err != nil {
				log.Log.Error(err.Error())
			}

			log.Log.Info("capture.main.HandleRecordStream(continuous): Recording finished: file save: " + name)

			// Cleanup muxer
			file.Close()
			file = nil

			// Check if need to convert to fragmented using bento
			if config.Capture.Fragmented == "true" && config.Capture.FragmentedDuration > 0 {
				utils.CreateFragmentedMP4(fullName, config.Capture.FragmentedDuration)
			}
		}

		log.Log.Debug("capture.main.HandleRecordStream(): finished")
	}
}
