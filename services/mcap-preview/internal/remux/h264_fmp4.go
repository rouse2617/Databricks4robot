// Package remux turns a sequence of H.264 Annex-B access units into an fMP4
// stream (ftyp+moov, then a series of moof+mdat fragments) suitable for
// progressive playback in an HTML5 <video> element.
//
// The implementation deliberately does NOT decode video: it only repackages
// already-encoded bytes, so it stays pure-Go and CPU-cheap.
//
// Limitations of this iteration:
//   - Single video track only.
//   - SPS/PPS captured once at WriteInit time (we currently do not emit
//     in-band SPS/PPS in fragments). Mid-stream parameter set changes will
//     therefore be ignored; that's acceptable while we only support
//     `foxglove.CompressedVideo` short clips.
//   - The caller is responsible for passing samples in decode order, with
//     keyframes correctly flagged. Fragments must start on a keyframe.
package remux

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/hevc"
	"github.com/Eyevinn/mp4ff/mp4"
)

var annexBStartCode4 = []byte{0, 0, 0, 1}

// Sample is a single decoded-time-stamped access unit in Annex-B form.
type Sample struct {
	// AnnexB carries one access unit as Annex-B byte stream
	// (start codes 0x000001/0x00000001 prefixed). It may include SPS/PPS
	// in addition to the VCL NALU(s) — those are stripped on conversion.
	AnnexB []byte
	// DurationTicks is the duration of this sample in the Remuxer
	// timescale (90 kHz unless overridden).
	DurationTicks uint32
	// IsKeyframe is true for IDR access units. The first sample in any
	// fragment MUST be a keyframe.
	IsKeyframe bool
}

// Remuxer is a stateful fMP4 muxer for a single H.264 video track. It is not
// safe for concurrent use.
type Remuxer struct {
	codec          string
	timescale      uint32
	trackID        uint32
	seqNumber      uint32
	baseDecodeTime uint64
	initSent       bool
}

// NewRemuxer creates a Remuxer with the given media timescale. The conventional
// value for 90 kHz video is 90000.
func NewRemuxer(timescale uint32) *Remuxer {
	return NewRemuxerForCodec("h264", timescale)
}

// NewRemuxerForCodec creates a Remuxer for h264 or h265/hevc.
func NewRemuxerForCodec(codec string, timescale uint32) *Remuxer {
	if timescale == 0 {
		timescale = 90000
	}
	c := strings.ToLower(codec)
	if c == "hevc" {
		c = "h265"
	}
	if c != "h265" {
		c = "h264"
	}
	return &Remuxer{codec: c, timescale: timescale, trackID: 1}
}

// TrackID returns the (single) video track id used by this Remuxer.
func (r *Remuxer) TrackID() uint32 { return r.trackID }

// Timescale returns the configured media timescale.
func (r *Remuxer) Timescale() uint32 { return r.timescale }

// WriteInit emits the ftyp+moov init segment built from the supplied SPS/PPS
// (one of each is sufficient — multiple are accepted for completeness).
// It must be called exactly once before WriteSegment.
// For H.265 use WriteInitHEVC.
func (r *Remuxer) WriteInit(w io.Writer, sps, pps [][]byte) error {
	if r.codec != "h264" {
		return errors.New("remux: WriteInit (AVC) called for non-h264 codec")
	}
	if r.initSent {
		return errors.New("remux: init already sent")
	}
	if len(sps) == 0 || len(pps) == 0 {
		return errors.New("remux: SPS and PPS required")
	}
	init := mp4.CreateEmptyInit()
	trak := init.AddEmptyTrack(r.timescale, "video", "und")
	if err := trak.SetAVCDescriptor("avc1", sps, pps, true); err != nil {
		return fmt.Errorf("remux: SetAVCDescriptor: %w", err)
	}
	if err := init.Encode(w); err != nil {
		return fmt.Errorf("remux: encode init: %w", err)
	}
	r.initSent = true
	return nil
}

// WriteInitHEVC emits ftyp+moov for an HEVC video track.
func (r *Remuxer) WriteInitHEVC(w io.Writer, vps, sps, pps [][]byte) error {
	if r.codec != "h265" {
		return errors.New("remux: WriteInitHEVC called for non-h265 codec")
	}
	if r.initSent {
		return errors.New("remux: init already sent")
	}
	if len(vps) == 0 || len(sps) == 0 || len(pps) == 0 {
		return errors.New("remux: VPS/SPS/PPS required for h265")
	}
	init := mp4.CreateEmptyInit()
	trak := init.AddEmptyTrack(r.timescale, "video", "und")
	// Use hev1 for broader browser compatibility when parameter sets may
	// also appear in-band in samples.
	if err := trak.SetHEVCDescriptor("hev1", vps, sps, pps, nil, true); err != nil {
		return fmt.Errorf("remux: SetHEVCDescriptor: %w", err)
	}
	if err := init.Encode(w); err != nil {
		return fmt.Errorf("remux: encode init: %w", err)
	}
	r.initSent = true
	return nil
}

// WriteSegment emits a single moof+mdat media segment carrying samples in
// decode order. samples[0].IsKeyframe must be true.
func (r *Remuxer) WriteSegment(w io.Writer, samples []Sample) error {
	if !r.initSent {
		return errors.New("remux: WriteInit must be called first")
	}
	if len(samples) == 0 {
		return nil
	}
	if !samples[0].IsKeyframe {
		return errors.New("remux: fragment must start with a keyframe")
	}

	r.seqNumber++
	seg := mp4.NewMediaSegment()
	frag, err := mp4.CreateFragment(r.seqNumber, r.trackID)
	if err != nil {
		return fmt.Errorf("remux: CreateFragment: %w", err)
	}
	seg.AddFragment(frag)

	dtAccum := r.baseDecodeTime
	for _, s := range samples {
		avccBytes := annexBToAVCC(s.AnnexB)
		avccBytes = stripParameterSetsByCodec(avccBytes, r.codec)

		flags := mp4.SampleFlags{}
		if s.IsKeyframe {
			flags.SampleDependsOn = 2
		} else {
			flags.SampleIsNonSync = true
			flags.SampleDependsOn = 1
		}

		fs := mp4.FullSample{
			Sample: mp4.Sample{
				Flags: flags.Encode(),
				Dur:   s.DurationTicks,
				Size:  uint32(len(avccBytes)),
			},
			DecodeTime: dtAccum,
			Data:       avccBytes,
		}
		if err := frag.AddFullSampleToTrack(fs, r.trackID); err != nil {
			return fmt.Errorf("remux: add sample: %w", err)
		}
		dtAccum += uint64(s.DurationTicks)
	}
	r.baseDecodeTime = dtAccum

	if err := seg.Encode(w); err != nil {
		return fmt.Errorf("remux: encode segment: %w", err)
	}
	return nil
}

// ExtractSPSPPS pulls the first SPS / PPS NALUs out of an Annex-B byte stream.
// Returns ok=false if either is missing.
func ExtractSPSPPS(annexB []byte) (sps, pps []byte, ok bool) {
	spss, ppss := avc.GetParameterSetsFromByteStream(annexB)
	if len(spss) == 0 || len(ppss) == 0 {
		return nil, nil, false
	}
	return spss[0], ppss[0], true
}

// IsAnnexBKeyframe reports whether the Annex-B access unit contains an IDR
// VCL NALU. Convenience wrapper to keep callers free of avc package imports.
func IsAnnexBKeyframe(annexB []byte) bool {
	for _, nalu := range avc.ExtractNalusFromByteStream(annexB) {
		if avc.GetNaluType(nalu[0]) == avc.NALU_IDR {
			return true
		}
	}
	return false
}

// IsAnnexBKeyframeHEVC reports whether the Annex-B access unit contains an
// HEVC IRAP NALU (types 16..23).
func IsAnnexBKeyframeHEVC(annexB []byte) bool {
	for _, nalu := range avc.ExtractNalusFromByteStream(annexB) {
		if len(nalu) < 2 {
			continue
		}
		nt := hevc.GetNaluType(nalu[0])
		// Be conservative: only IDR pictures are guaranteed random access.
		// CRA/BLA may still depend on previous decoded pictures on some
		// encoders, which causes browser decoders to fail on seek/segment
		// boundaries.
		if nt == hevc.NALU_IDR_W_RADL || nt == hevc.NALU_IDR_N_LP {
			return true
		}
	}
	return false
}

// NormalizeToAnnexB ensures frame payload is in Annex-B start-code form.
// Some producers publish length-prefixed NAL units (AVCC/hvcC packet style)
// even though the schema expects Annex-B.
func NormalizeToAnnexB(frame []byte) []byte {
	if len(frame) < 4 {
		return frame
	}
	if hasAnnexBPrefix(frame) {
		return frame
	}
	for boxSize := 4; boxSize >= 1; boxSize-- {
		if out, ok := packetToAnnexB(frame, boxSize); ok {
			return out
		}
	}
	return frame
}

func hasAnnexBPrefix(b []byte) bool {
	if len(b) >= 4 && b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 1 {
		return true
	}
	return len(b) >= 3 && b[0] == 0 && b[1] == 0 && b[2] == 1
}

func packetToAnnexB(frame []byte, lengthFieldSize int) ([]byte, bool) {
	if len(frame) <= lengthFieldSize {
		return nil, false
	}
	out := make([]byte, 0, len(frame)+64)
	pos := 0
	packets := 0
	for {
		if pos+lengthFieldSize > len(frame) {
			break
		}
		nLen := int(readUIntNBE(frame[pos:pos+lengthFieldSize], lengthFieldSize))
		pos += lengthFieldSize
		if nLen <= 0 || pos+nLen > len(frame) {
			return nil, false
		}
		out = append(out, annexBStartCode4...)
		out = append(out, frame[pos:pos+nLen]...)
		pos += nLen
		packets++
	}
	if packets == 0 || pos != len(frame) {
		return nil, false
	}
	return out, true
}

func readUIntNBE(b []byte, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = (v << 8) | uint32(b[i])
	}
	return v
}

// annexBToAVCC converts an Annex-B byte stream containing one or more NALUs
// into AVCC length-prefixed form (4-byte big-endian length + NALU bytes).
func annexBToAVCC(annexB []byte) []byte {
	nalus := avc.ExtractNalusFromByteStream(annexB)
	if len(nalus) == 0 {
		return nil
	}
	total := 0
	for _, n := range nalus {
		total += 4 + len(n)
	}
	out := make([]byte, 0, total)
	for _, n := range nalus {
		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(n)))
		out = append(out, lenBuf[:]...)
		out = append(out, n...)
	}
	return out
}

func stripParameterSetsByCodec(avccBytes []byte, codec string) []byte {
	if codec == "h265" {
		// Keep VPS/SPS/PPS in-band for HEVC. Several browser decode pipelines
		// are stricter with fragmented streams and rely on in-sample parameter
		// sets even when hvcC is present.
		return avccBytes
	}
	return stripAVCParameterSets(avccBytes)
}

// stripParameterSets removes any SPS / PPS / AUD NALUs from an AVCC byte
// stream, leaving only VCL NALUs (and SEI). Parameter sets live in the
// moov/avcC and would only confuse a chunked-MP4 decoder if duplicated here.
func stripAVCParameterSets(avccBytes []byte) []byte {
	if len(avccBytes) == 0 {
		return avccBytes
	}
	out := avccBytes[:0:len(avccBytes)]
	out = make([]byte, 0, len(avccBytes))
	pos := 0
	for pos+4 <= len(avccBytes) {
		nLen := int(binary.BigEndian.Uint32(avccBytes[pos : pos+4]))
		end := pos + 4 + nLen
		if end > len(avccBytes) || nLen <= 0 {
			break
		}
		hdr := avccBytes[pos+4]
		nt := avc.GetNaluType(hdr)
		if nt != avc.NALU_SPS && nt != avc.NALU_PPS && nt != avc.NALU_AUD {
			out = append(out, avccBytes[pos:end]...)
		}
		pos = end
	}
	if len(out) == 0 {
		// Nothing left after stripping — return original to avoid producing
		// a zero-length sample. The caller filters such samples upstream.
		return avccBytes
	}
	return out
}

// stripHEVCParameterSets removes VPS/SPS/PPS/AUD from a HEVC AVCC stream.
func stripHEVCParameterSets(avccBytes []byte) []byte {
	if len(avccBytes) == 0 {
		return avccBytes
	}
	out := make([]byte, 0, len(avccBytes))
	pos := 0
	for pos+4 <= len(avccBytes) {
		nLen := int(binary.BigEndian.Uint32(avccBytes[pos : pos+4]))
		end := pos + 4 + nLen
		if end > len(avccBytes) || nLen <= 0 {
			break
		}
		nalu := avccBytes[pos+4 : end]
		keep := true
		if len(nalu) >= 2 {
			nt := hevc.GetNaluType(nalu[0])
			if nt == hevc.NALU_VPS || nt == hevc.NALU_SPS || nt == hevc.NALU_PPS || nt == hevc.NALU_AUD {
				keep = false
			}
		}
		if keep {
			out = append(out, avccBytes[pos:end]...)
		}
		pos = end
	}
	if len(out) == 0 {
		return avccBytes
	}
	return out
}

// ExtractVPSPPSHEVC pulls the first VPS/SPS/PPS NALUs from an Annex-B stream.
func ExtractVPSPPSHEVC(annexB []byte) (vps, sps, pps []byte, ok bool) {
	vpss, spss, ppss := hevc.GetParameterSetsFromByteStream(annexB)
	if len(vpss) == 0 || len(spss) == 0 || len(ppss) == 0 {
		return nil, nil, nil, false
	}
	return vpss[0], spss[0], ppss[0], true
}
