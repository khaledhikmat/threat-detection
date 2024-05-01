package config

type Capturer struct {
	MaxCameras         int    `json:"maxCameras"`
	RecordingsFolder   string `json:"recordingsFolder"`
	StorageDestination string `json:"storageDestination"`
}
