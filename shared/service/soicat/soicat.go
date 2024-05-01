package soicat

import (
	"time"

	"github.com/guregu/null"
)

func New() IService {
	return &soicat{}
}

type soicat struct {
}

func (s *soicat) UncapturedCameras() ([]Camera, error) {
	return []Camera{
		{
			ID:                 "100",
			Name:               "Camera1",
			RtspURL:            "rtsp://admin:gooze_bumbs@192.168.1.206:554/cam/realmonitor?channel=1&subtype=0",
			IsAnalytics:        true,
			Capturer:           "pod-1",
			LastHeartBeat:      null.TimeFrom(time.Now().Add(-5 * time.Minute)),
			CaptureWidth:       500,
			CaptureHeight:      500,
			MaxLengthRecording: 3,
			Timezone:           "",
			RecordingsFolder:   "./data/recordings",
		},
	}, nil
}

func (s *soicat) Finalize() {
}
