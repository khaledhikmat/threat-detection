package capturer

// #cgo pkg-config: libavcodec libavutil libswscale
// #include <libavcodec/avcodec.h>
// #include <libavutil/imgutils.h>
// #include <libswscale/swscale.h>
import "C"

import (
	"context"
	"errors"
	"fmt"
	"image"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph265"
	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/pkg/codecs/h265"
	"github.com/khaledhikmat/threat-detection/shared/service/soicat"
	"github.com/pion/rtp"
)

func NewRTSPClient(rtspURL string) *Golibrtsp {
	return &Golibrtsp{
		URL: rtspURL,
	}
}

// Implements the RTSPClient interface.
type Golibrtsp struct {
	RTSPClient
	URL string

	Client            gortsplib.Client
	VideoDecoderMutex *sync.Mutex

	VideoH264Index        int8
	VideoH264Media        *description.Media
	VideoH264Forma        *format.H264
	VideoH264Decoder      *rtph264.Decoder
	VideoH264FrameDecoder *Decoder

	VideoH265Index        int8
	VideoH265Media        *description.Media
	VideoH265Forma        *format.H265
	VideoH265Decoder      *rtph265.Decoder
	VideoH265FrameDecoder *Decoder

	Streams []Stream
}

// Connect to the RTSP server.
func (g *Golibrtsp) Connect(_ context.Context) error {

	transport := gortsplib.TransportTCP
	g.Client = gortsplib.Client{
		RequestBackChannels: false,
		Transport:           &transport,
	}

	// parse URL
	u, err := base.ParseURL(g.URL)
	if err != nil {
		return err
	}

	// connect to the server
	err = g.Client.Start(u.Scheme, u.Host)
	if err != nil {
		return err
	}

	// find published medias
	desc, _, err := g.Client.Describe(u)
	if err != nil {
		return err
	}

	// Iniatlize the mutex.
	g.VideoDecoderMutex = &sync.Mutex{}

	// Setup H264
	h264Err := g.setupH264(desc)

	// Setup H265
	h265Err := g.setupH265(desc)

	// If both codecs are not supported, return an error
	if h264Err != nil && h265Err != nil {
		return fmt.Errorf("both H264 and H265 are not supported: %v and %v", h264Err.Error(), h265Err.Error())
	}

	return nil
}

func (g *Golibrtsp) setupH264(desc *description.Session) error {
	// find the H264 media and format
	var formaH264 *format.H264
	mediH264 := desc.FindFormat(&formaH264)
	g.VideoH264Media = mediH264
	g.VideoH264Forma = formaH264
	if mediH264 == nil {
		return fmt.Errorf("capture.golibrtsp.Connect(H264) - video media not found")
	}

	// setup a video media
	_, err := g.Client.Setup(desc.BaseURL, mediH264, 0, 0)
	if err != nil {
		return err
	}

	// Get SPS and PPS from the SDP
	// Calculate the width and height of the video
	var sps h264.SPS
	errSPS := sps.Unmarshal(formaH264.SPS)
	// It might be that the SPS is not available yet, so we'll proceed,
	// but try to fetch it later on.
	if errSPS != nil {
		fmt.Printf("capture.golibrtsp.Connect(H264): %v\n", errSPS)
		g.Streams = append(g.Streams, Stream{
			Name:          formaH264.Codec(),
			IsVideo:       true,
			IsAudio:       false,
			SPS:           []byte{},
			PPS:           []byte{},
			Width:         0,
			Height:        0,
			FPS:           0,
			IsBackChannel: false,
		})
	} else {
		g.Streams = append(g.Streams, Stream{
			Name:          formaH264.Codec(),
			IsVideo:       true,
			IsAudio:       false,
			SPS:           formaH264.SPS,
			PPS:           formaH264.PPS,
			Width:         sps.Width(),
			Height:        sps.Height(),
			FPS:           sps.FPS(),
			IsBackChannel: false,
		})
	}

	// Set the index for the video
	g.VideoH264Index = int8(len(g.Streams)) - 1

	// setup RTP/H264 -> H264 decoder
	rtpDec, err := formaH264.CreateDecoder()
	if err != nil {
		return err
	}
	g.VideoH264Decoder = rtpDec

	// setup H264 -> raw frames decoder
	frameDec, err := newDecoder("H264")
	if err != nil {
		return err
	}
	g.VideoH264FrameDecoder = frameDec

	return nil
}

func (g *Golibrtsp) setupH265(desc *description.Session) error {
	// find the H265 media and format
	var formaH265 *format.H265
	mediH265 := desc.FindFormat(&formaH265)
	g.VideoH265Media = mediH265
	g.VideoH265Forma = formaH265
	if mediH265 == nil {
		return fmt.Errorf("capture.golibrtsp.Connect(H265) - video media not found")
	}

	// setup a video media
	_, err := g.Client.Setup(desc.BaseURL, mediH265, 0, 0)
	if err != nil {
		return err
	}

	// Get SPS from the SDP
	// Calculate the width and height of the video
	var sps h265.SPS
	err = sps.Unmarshal(formaH265.SPS)
	if err != nil {
		return err
	}

	g.Streams = append(g.Streams, Stream{
		Name:          formaH265.Codec(),
		IsVideo:       true,
		IsAudio:       false,
		SPS:           formaH265.SPS,
		PPS:           formaH265.PPS,
		VPS:           formaH265.VPS,
		Width:         sps.Width(),
		Height:        sps.Height(),
		FPS:           sps.FPS(),
		IsBackChannel: false,
	})

	// Set the index for the video
	g.VideoH265Index = int8(len(g.Streams)) - 1

	// setup RTP/H265 -> H265 decoder
	rtpDec, err := formaH265.CreateDecoder()
	if err != nil {
		return err
	}
	g.VideoH265Decoder = rtpDec

	// setup H265 -> raw frames decoder
	frameDec, err := newDecoder("H265")
	if err != nil {
		return err
	}
	g.VideoH265FrameDecoder = frameDec

	return nil
}

// Start the RTSP client, and start reading packets.
func (g *Golibrtsp) Start(_ context.Context, errorsStream chan interface{}, packetsStream chan Packet, camera soicat.Camera) error {
	fmt.Printf("capture.golibrtsp.Start(): started\n")

	// called when a video RTP packet arrives for H264
	var filteredAU [][]byte
	if g.VideoH264Media != nil && g.VideoH264Forma != nil {
		g.Client.OnPacketRTP(g.VideoH264Media, g.VideoH264Forma, func(rtppkt *rtp.Packet) {

			if len(rtppkt.Payload) > 0 {

				// decode timestamp
				pts, ok := g.Client.PacketPTS(g.VideoH264Media, rtppkt)
				if !ok {
					errorsStream <- fmt.Errorf("capture.golibrtsp.Start(): %s", "unable to get PTS")
					return
				}

				// Extract access units from RTP packets
				// We need to do this, because the decoder expects a full
				// access unit. Once we have a full access unit, we can
				// decode it, and know if it's a keyframe or not.
				au, errDecode := g.VideoH264Decoder.Decode(rtppkt)
				if errDecode != nil {
					if errDecode != rtph264.ErrNonStartingPacketAndNoPrevious && errDecode != rtph264.ErrMorePacketsNeeded {
						errorsStream <- fmt.Errorf("capture.golibrtsp.Start(): %v", errDecode)
					}
					return
				}

				// We'll need to read out a few things.
				// prepend an AUD. This is required by some players
				filteredAU = [][]byte{
					{byte(h264.NALUTypeAccessUnitDelimiter), 240},
				}

				// Check if we have a keyframe.
				nonIDRPresent := false
				idrPresent := false

				for _, nalu := range au {
					typ := h264.NALUType(nalu[0] & 0x1F)
					switch typ {
					case h264.NALUTypeAccessUnitDelimiter:
						continue
					case h264.NALUTypeIDR:
						idrPresent = true
					case h264.NALUTypeNonIDR:
						nonIDRPresent = true
					case h264.NALUTypeSPS:
						// Read out sps
						var sps h264.SPS
						errSPS := sps.Unmarshal(nalu)
						if errSPS == nil {
							// Get width
							g.Streams[g.VideoH264Index].Width = sps.Width()
							camera.CaptureWidth = sps.Width()

							// Get height
							g.Streams[g.VideoH264Index].Height = sps.Height()
							camera.CaptureHeight = sps.Height()
							// Get FPS
							g.Streams[g.VideoH264Index].FPS = sps.FPS()
							g.VideoH264Forma.SPS = nalu
						}
					case h264.NALUTypePPS:
						// Read out pps
						g.VideoH264Forma.PPS = nalu
					}
					filteredAU = append(filteredAU, nalu)
				}

				if len(filteredAU) <= 1 || (!nonIDRPresent && !idrPresent) {
					return
				}

				// Convert to packet.
				enc, err := h264.AnnexBMarshal(filteredAU)
				if err != nil {
					errorsStream <- fmt.Errorf("capture.golibrtsp.Start(): %v", err)
					return
				}

				pkt := Packet{
					IsKeyFrame:      idrPresent,
					Packet:          rtppkt,
					Data:            enc,
					Time:            pts,
					CompositionTime: pts,
					Idx:             g.VideoH264Index,
					IsVideo:         true,
					IsAudio:         false,
					Codec:           "H264",
				}

				pkt.Data = pkt.Data[4:]
				if pkt.IsKeyFrame {
					annexbNALUStartCode := func() []byte { return []byte{0x00, 0x00, 0x00, 0x01} }
					pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
					pkt.Data = append(g.VideoH264Forma.PPS, pkt.Data...)
					pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
					pkt.Data = append(g.VideoH264Forma.SPS, pkt.Data...)
					pkt.Data = append(annexbNALUStartCode(), pkt.Data...)
				}

				packetsStream <- pkt
			}

		})
	}

	// called when a video RTP packet arrives for H265
	if g.VideoH265Media != nil && g.VideoH265Forma != nil {
		g.Client.OnPacketRTP(g.VideoH265Media, g.VideoH265Forma, func(rtppkt *rtp.Packet) {

			if len(rtppkt.Payload) > 0 {

				// decode timestamp
				pts, ok := g.Client.PacketPTS(g.VideoH265Media, rtppkt)
				if !ok {
					errorsStream <- fmt.Errorf("capture.golibrtsp.Start(): %s", "unable to get PTS")
					return
				}

				// Extract access units from RTP packets
				// We need to do this, because the decoder expects a full
				// access unit. Once we have a full access unit, we can
				// decode it, and know if it's a keyframe or not.
				au, errDecode := g.VideoH265Decoder.Decode(rtppkt)
				if errDecode != nil {
					if errDecode != rtph265.ErrNonStartingPacketAndNoPrevious && errDecode != rtph265.ErrMorePacketsNeeded {
						errorsStream <- fmt.Errorf("capture.golibrtsp.Start(): %v", errDecode)
					}
					return
				}

				filteredAU = [][]byte{
					{byte(h265.NALUType_AUD_NUT) << 1, 1, 0x50},
				}

				isRandomAccess := false
				for _, nalu := range au {
					typ := h265.NALUType((nalu[0] >> 1) & 0b111111)
					switch typ {
					/*case h265.NALUType_VPS_NUT:
					continue*/
					case h265.NALUType_SPS_NUT:
						continue
					case h265.NALUType_PPS_NUT:
						continue
					case h265.NALUType_AUD_NUT:
						continue
					case h265.NALUType_IDR_W_RADL, h265.NALUType_IDR_N_LP, h265.NALUType_CRA_NUT:
						isRandomAccess = true
					}
					filteredAU = append(filteredAU, nalu)
				}

				au = filteredAU

				if len(au) <= 1 {
					return
				}

				// add VPS, SPS and PPS before random access access unit
				if isRandomAccess {
					au = append([][]byte{
						g.VideoH265Forma.VPS,
						g.VideoH265Forma.SPS,
						g.VideoH265Forma.PPS}, au...)
				}

				enc, err := h264.AnnexBMarshal(au)
				if err != nil {
					errorsStream <- fmt.Errorf("capture.golibrtsp.Start(): %v", err)
					return
				}

				pkt := Packet{
					IsKeyFrame:      isRandomAccess,
					Packet:          rtppkt,
					Data:            enc,
					Time:            pts,
					CompositionTime: pts,
					Idx:             g.VideoH265Index,
					IsVideo:         true,
					IsAudio:         false,
					Codec:           "H265",
				}

				packetsStream <- pkt
			}

		})
	}

	// Wait for a second, so we can be sure the stream is playing.
	time.Sleep(1 * time.Second)

	// Play the stream.
	_, err := g.Client.Play(nil)
	if err != nil {
		return err
	}

	return nil
}

// Decode a packet to an image.
func (g *Golibrtsp) DecodePacket(pkt Packet) (image.YCbCr, error) {
	var img image.YCbCr
	var err error
	g.VideoDecoderMutex.Lock()
	if len(pkt.Data) == 0 {
		err = errors.New("TSPClient(Golibrtsp).DecodePacket(): empty frame")
	} else if g.VideoH264Decoder != nil {
		img, err = g.VideoH264FrameDecoder.decode(pkt.Data)
	} else if g.VideoH265Decoder != nil {
		img, err = g.VideoH265FrameDecoder.decode(pkt.Data)
	} else {
		err = errors.New("TSPClient(Golibrtsp).DecodePacket(): no decoder found, might already be closed")
	}
	g.VideoDecoderMutex.Unlock()
	if err != nil {
		fmt.Printf("capture.golibrtsp.DecodePacket(): %v\n", err)
		return image.YCbCr{}, err
	}
	if img.Bounds().Empty() {
		fmt.Printf("capture.golibrtsp.DecodePacket(): empty frame\n")
		return image.YCbCr{}, errors.New("Empty image")
	}
	return img, nil
}

// Decode a packet to a Gray image.
func (g *Golibrtsp) DecodePacketRaw(pkt Packet) (image.Gray, error) {
	var img image.Gray
	var err error
	g.VideoDecoderMutex.Lock()
	if len(pkt.Data) == 0 {
		err = errors.New("capture.golibrtsp.DecodePacketRaw(): empty frame")
	} else if g.VideoH264Decoder != nil {
		img, err = g.VideoH264FrameDecoder.decodeRaw(pkt.Data)
	} else if g.VideoH265Decoder != nil {
		img, err = g.VideoH265FrameDecoder.decodeRaw(pkt.Data)
	} else {
		err = errors.New("capture.golibrtsp.DecodePacketRaw(): no decoder found, might already be closed")
	}
	g.VideoDecoderMutex.Unlock()
	if err != nil {
		fmt.Printf("capture.golibrtsp.DecodePacketRaw(): %v\n", err)
		return image.Gray{}, err
	}
	if img.Bounds().Empty() {
		fmt.Printf("capture.golibrtsp.DecodePacketRaw(): empty image\n")
		return image.Gray{}, errors.New("Empty image")
	}

	// Do a deep copy of the image
	imgDeepCopy := image.NewGray(img.Bounds())
	imgDeepCopy.Stride = img.Stride
	copy(imgDeepCopy.Pix, img.Pix)

	return *imgDeepCopy, err
}

// Get a list of streams from the RTSP server.
func (g *Golibrtsp) GetStreams() ([]Stream, error) {
	return g.Streams, nil
}

// Get a list of video streams from the RTSP server.
func (g *Golibrtsp) GetVideoStreams() ([]Stream, error) {
	var videoStreams []Stream
	for _, stream := range g.Streams {
		if stream.IsVideo {
			videoStreams = append(videoStreams, stream)
		}
	}
	return videoStreams, nil
}

// Close the connection to the RTSP server.
func (g *Golibrtsp) Close() error {
	// Close the demuxer.
	g.Client.Close()
	if g.VideoH264Decoder != nil {
		g.VideoH264FrameDecoder.Close()
	}
	if g.VideoH265FrameDecoder != nil {
		g.VideoH265FrameDecoder.Close()
	}
	return nil
}

func frameData(frame *C.AVFrame) **C.uint8_t {
	return (**C.uint8_t)(unsafe.Pointer(&frame.data[0]))
}

func frameLineSize(frame *C.AVFrame) *C.int {
	return (*C.int)(unsafe.Pointer(&frame.linesize[0]))
}

// h264Decoder is a wrapper around FFmpeg's H264 decoder.
type Decoder struct {
	codecCtx *C.AVCodecContext
	srcFrame *C.AVFrame
}

// newH264Decoder allocates a new h264Decoder.
func newDecoder(codecName string) (*Decoder, error) {
	codec := C.avcodec_find_decoder(C.AV_CODEC_ID_H264)
	if codecName == "H265" {
		codec = C.avcodec_find_decoder(C.AV_CODEC_ID_H265)
	}
	if codec == nil {
		return nil, fmt.Errorf("avcodec_find_decoder() failed")
	}

	codecCtx := C.avcodec_alloc_context3(codec)
	if codecCtx == nil {
		return nil, fmt.Errorf("avcodec_alloc_context3() failed")
	}

	res := C.avcodec_open2(codecCtx, codec, nil)
	if res < 0 {
		C.avcodec_close(codecCtx)
		return nil, fmt.Errorf("avcodec_open2() failed")
	}

	srcFrame := C.av_frame_alloc()
	if srcFrame == nil {
		C.avcodec_close(codecCtx)
		return nil, fmt.Errorf("av_frame_alloc() failed")
	}

	return &Decoder{
		codecCtx: codecCtx,
		srcFrame: srcFrame,
	}, nil
}

// close closes the decoder.
func (d *Decoder) Close() {
	if d.srcFrame != nil {
		C.av_frame_free(&d.srcFrame)
	}
	C.av_frame_free(&d.srcFrame)
	C.avcodec_close(d.codecCtx)
}

func (d *Decoder) decode(nalu []byte) (image.YCbCr, error) {
	nalu = append([]uint8{0x00, 0x00, 0x00, 0x01}, []uint8(nalu)...)

	// send NALU to decoder
	var avPacket C.AVPacket
	avPacket.data = (*C.uint8_t)(C.CBytes(nalu))
	defer C.free(unsafe.Pointer(avPacket.data))
	avPacket.size = C.int(len(nalu))
	res := C.avcodec_send_packet(d.codecCtx, &avPacket)
	if res < 0 {
		return image.YCbCr{}, nil
	}

	// receive frame if available
	res = C.avcodec_receive_frame(d.codecCtx, d.srcFrame)
	if res < 0 {
		return image.YCbCr{}, nil
	}

	if res == 0 {
		fr := d.srcFrame
		w := int(fr.width)
		h := int(fr.height)
		ys := int(fr.linesize[0])
		cs := int(fr.linesize[1])

		return image.YCbCr{
			Y:              fromCPtr(unsafe.Pointer(fr.data[0]), ys*h),
			Cb:             fromCPtr(unsafe.Pointer(fr.data[1]), cs*h/2),
			Cr:             fromCPtr(unsafe.Pointer(fr.data[2]), cs*h/2),
			YStride:        ys,
			CStride:        cs,
			SubsampleRatio: image.YCbCrSubsampleRatio420,
			Rect:           image.Rect(0, 0, w, h),
		}, nil
	}

	return image.YCbCr{}, nil
}

func (d *Decoder) decodeRaw(nalu []byte) (image.Gray, error) {
	nalu = append([]uint8{0x00, 0x00, 0x00, 0x01}, []uint8(nalu)...)

	// send NALU to decoder
	var avPacket C.AVPacket
	avPacket.data = (*C.uint8_t)(C.CBytes(nalu))
	defer C.free(unsafe.Pointer(avPacket.data))
	avPacket.size = C.int(len(nalu))
	res := C.avcodec_send_packet(d.codecCtx, &avPacket)
	if res < 0 {
		return image.Gray{}, nil
	}

	// receive frame if available
	res = C.avcodec_receive_frame(d.codecCtx, d.srcFrame)
	if res < 0 {
		return image.Gray{}, nil
	}

	if res == 0 {
		fr := d.srcFrame
		w := int(fr.width)
		h := int(fr.height)
		ys := int(fr.linesize[0])

		return image.Gray{
			Pix:    fromCPtr(unsafe.Pointer(fr.data[0]), w*h),
			Stride: ys,
			Rect:   image.Rect(0, 0, w, h),
		}, nil
	}

	return image.Gray{}, nil
}

func fromCPtr(buf unsafe.Pointer, size int) (ret []uint8) {
	hdr := (*reflect.SliceHeader)((unsafe.Pointer(&ret)))
	hdr.Cap = size
	hdr.Len = size
	hdr.Data = uintptr(buf)
	return
}

func FindPCMU(desc *description.Session, isBackChannel bool) (*format.G711, *description.Media) {
	for _, media := range desc.Medias {
		if media.IsBackChannel == isBackChannel {
			for _, forma := range media.Formats {
				if g711, ok := forma.(*format.G711); ok {
					if g711.MULaw {
						return g711, media
					}
				}
			}
		}
	}
	return nil, nil
}
