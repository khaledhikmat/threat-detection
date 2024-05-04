package equates

const (
	ThreatDetectionSecrets    = "threat-detection-secrets"
	ThreatDetectionStateStore = "threat-detection-statestore" // name must match config/redis-statestore.yaml
	ThreatDetectionPubSub     = "threat-detection-pubsub"     // name must match config/redis-pubsub.yaml
	RecordingsTopic           = "recordings-topic"
	NotificationsTopic        = "notifications-topic"
	MetadataTopic             = "metadata-topic"
)

type RecordingClip struct {
	ID             string `json:"id"`
	LocalReference string `json:"localReference"`
	CloudReference string `json:"cloudReference"`
	Capturer       string `json:"capturer"`
	Camera         string `json:"camera"`
	Frames         int    `json:"frames"`
}
