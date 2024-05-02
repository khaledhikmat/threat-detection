package soicat

import (
	"fmt"
	"time"

	"github.com/guregu/null"
)

var cameras = []Camera{
	{
		ID:                 "100",
		Name:               "Camera1",
		RtspURL:            "rtsp://admin:gooze_bumbs@192.168.1.206:554/cam/realmonitor?channel=1&subtype=0",
		IsAnalytics:        true,
		Capturer:           "dead-capturer",
		LastHeartBeat:      null.TimeFrom(time.Now().Add(-5 * time.Minute)),
		CaptureWidth:       500,
		CaptureHeight:      500,
		MaxLengthRecording: 3,
		Timezone:           "",
		RecordingsFolder:   "./data/recordings",
	},
}

func New() IService {
	return &soicat{}
}

type soicat struct {
}

func (s *soicat) UpdateCamera(c Camera) error {
	for idx, camera := range cameras {
		if camera.Name == c.Name {
			ptr := &cameras[idx]
			(*ptr).Capturer = c.Capturer
			return nil
		}
	}

	return fmt.Errorf("could not find the camera %s", c.Name)
}

func (s *soicat) Cameras() ([]Camera, error) {
	return cameras, nil
}

func (s *soicat) Finalize() {
}
