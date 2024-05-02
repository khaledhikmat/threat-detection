package capturer

import "time"

type Capturer struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	LastHeartBeat time.Time `json:"lastHeartBeat"`
}
