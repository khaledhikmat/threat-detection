package soicat

import "github.com/guregu/null"

type Camera struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	RtspURL            string    `json:"rtspUrl"`
	IsAnalytics        bool      `json:"isAnalytics"`
	Capturer           string    `json:"capturer"`
	RecordingState     string    `json:"recordingState"` // recording, idle,
	ClientState        string    `json:"clientState"`    // started, stopped, paused
	LastHeartBeat      null.Time `json:"lastHeartBeat"`
	CaptureWidth       int       `json:"captureWidth"`
	CaptureHeight      int       `json:"captureHeight"`
	PreRecording       int64     `json:"preRecording"`
	MaxLengthRecording int64     `json:"maxLengthRecording"`
	Timezone           string    `json:"timezone"`
	RecordingsFolder   string    `json:"recordingsFolder"`
}
