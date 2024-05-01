package capturer

import (
	"time"

	"github.com/pion/rtp"
)

// Models are borrowed from: https://github.com/kerberos-io/agent/machinery

// Packet represents an RTP Packet
type Packet struct {
	Packet          *rtp.Packet
	IsAudio         bool          // packet is audio
	IsVideo         bool          // packet is video
	IsKeyFrame      bool          // video packet is key frame
	Idx             int8          // stream index in container format
	Codec           string        // codec name
	CompositionTime time.Duration // packet presentation time minus decode time for H264 B-Frame
	Time            time.Duration // packet decode time
	Data            []byte        // packet data
}

type Stream struct {
	// The name of the stream.
	Name string

	// The URL of the stream.
	URL string

	// Is the stream a video stream.
	IsVideo bool

	// Is the stream a audio stream.
	IsAudio bool

	// The width of the stream.
	Width int

	// The height of the stream.
	Height int

	// Num is the numerator of the framerate.
	Num int

	// Denum is the denominator of the framerate.
	Denum int

	// FPS is the framerate of the stream.
	FPS float64

	// For H264, this is the sps.
	SPS []byte

	// For H264, this is the pps.
	PPS []byte

	// For H265, this is the vps.
	VPS []byte

	// IsBackChannel is true if this stream is a back channel.
	IsBackChannel bool
}
