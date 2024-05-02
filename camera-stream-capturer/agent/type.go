package agent

import (
	"context"
	"image"

	"github.com/khaledhikmat/threat-detection/shared/service/soicat"
)

// RTSPClient is a interface that abstracts the RTSP client implementation.
type RTSPClient interface {
	// Connect to the RTSP server.
	Connect(ctx context.Context) error

	// Start the RTSP client, and start reading packets.
	Start(ctx context.Context, errorsStream chan interface{}, packetsStream chan Packet, camera soicat.Camera) error

	// Decode a packet into a image.
	DecodePacket(pkt Packet) (image.YCbCr, error)

	// Decode a packet into a image.
	DecodePacketRaw(pkt Packet) (image.Gray, error)

	// Pause the recordinhg/play to the RTSP server.
	Pause() error

	// Resume the recordinhg/play to the RTSP server.
	Resume() error

	// Close the connection to the RTSP server.
	Close() error

	// Get a list of streams from the RTSP server.
	GetStreams() ([]Stream, error)

	// Get a list of video streams from the RTSP server.
	GetVideoStreams() ([]Stream, error)
}
