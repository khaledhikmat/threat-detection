package soicat

import "github.com/guregu/null"

type Camera struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	RtspURL       string    `json:"rtspUrl"`
	IsAnalytics   bool      `json:"isAnalytics"`
	Capturer      string    `json:"capturer"`
	LastHeartBeat null.Time `json:"lastHeartBeat"`
}
