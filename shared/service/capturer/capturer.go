package capturer

import "time"

func New() IService {
	return &capturer{}
}

type capturer struct {
}

// Capturers return alive
func (s *capturer) Capturers() ([]Capturer, error) {
	return []Capturer{
		{
			ID:            "100",
			Name:          "capturer1",
			LastHeartBeat: time.Now().Add(-5 * time.Minute),
		},
	}, nil
}

func (s *capturer) Finalize() {
}
