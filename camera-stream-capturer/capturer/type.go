package capturer

import (
	"context"
	"image"

	"github.com/kerberos-io/agent/machinery/src/packets"

	"github.com/khaledhikmat/threat-detection/shared/service/config"
	"github.com/khaledhikmat/threat-detection/shared/service/soicat"
)

// RTSPClient is a interface that abstracts the RTSP client implementation.
type RTSPClient interface {
	// Connect to the RTSP server.
	Connect(ctx context.Context) error

	// Start the RTSP client, and start reading packets.
	Start(ctx context.Context, queue *packets.Queue, cfgsvc config.IService, camera soicat.Camera) error

	// Decode a packet into a image.
	DecodePacket(pkt packets.Packet) (image.YCbCr, error)

	// Decode a packet into a image.
	DecodePacketRaw(pkt packets.Packet) (image.Gray, error)

	// Close the connection to the RTSP server.
	Close() error

	// Get a list of streams from the RTSP server.
	GetStreams() ([]packets.Stream, error)

	// Get a list of video streams from the RTSP server.
	GetVideoStreams() ([]packets.Stream, error)
}
