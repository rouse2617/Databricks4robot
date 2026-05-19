package remux

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/hevc"
	"github.com/Eyevinn/mp4ff/mp4"
)

// Sample SPS / PPS taken from the upstream initcreator example
// (1280x720, baseline profile). Sufficient to populate avcC.
var (
	testSPSHex     = "67640020accac05005bb0169e0000003002000000c9c4c000432380008647c12401cb1c31380"
	testPPSHex     = "68b5df20"
	testHEVCVPSHex = "40010c01ffff022000000300b0000003000003007b18b024"
	testHEVCSPSHex = "420101022000000300b0000003000003007ba0078200887db6718b92448053888892cf24a69272c9124922dc91aa48fca223ff000100016a02020201"
	testHEVCPPSHex = "4401c0252f053240"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	return b
}

// makeAnnexB builds an access unit with an optional SPS/PPS prefix and a
// fake VCL NALU (IDR or non-IDR depending on `idr`). The VCL payload bytes
// are arbitrary — we only verify packaging, not video correctness.
func makeAnnexB(t *testing.T, withPS, idr bool) []byte {
	t.Helper()
	var b bytes.Buffer
	startCode := []byte{0, 0, 0, 1}
	if withPS {
		b.Write(startCode)
		b.Write(mustHex(t, testSPSHex))
		b.Write(startCode)
		b.Write(mustHex(t, testPPSHex))
	}
	b.Write(startCode)
	if idr {
		b.WriteByte(0x65) // forbidden_zero_bit=0, nal_ref_idc=11, type=5 (IDR)
	} else {
		b.WriteByte(0x41) // type=1 (non-IDR)
	}
	// payload — arbitrary 32 bytes that won't accidentally form a start code.
	for i := 0; i < 32; i++ {
		b.WriteByte(byte(0x10 + (i & 0x0f)))
	}
	return b.Bytes()
}

func makeHEVCAnnexB(t *testing.T, withPS, idr bool) []byte {
	t.Helper()
	var b bytes.Buffer
	startCode := []byte{0, 0, 0, 1}
	if withPS {
		b.Write(startCode)
		b.Write(mustHex(t, testHEVCVPSHex))
		b.Write(startCode)
		b.Write(mustHex(t, testHEVCSPSHex))
		b.Write(startCode)
		b.Write(mustHex(t, testHEVCPPSHex))
	}
	b.Write(startCode)
	if idr {
		b.Write([]byte{0x26, 0x01}) // type 19
	} else {
		b.Write([]byte{0x02, 0x01}) // type 1
	}
	for i := 0; i < 32; i++ {
		b.WriteByte(byte(0x20 + (i & 0x0f)))
	}
	return b.Bytes()
}

func TestExtractSPSPPS(t *testing.T) {
	au := makeAnnexB(t, true, true)
	sps, pps, ok := ExtractSPSPPS(au)
	if !ok {
		t.Fatal("ExtractSPSPPS returned !ok")
	}
	if !bytes.Equal(sps, mustHex(t, testSPSHex)) {
		t.Errorf("sps mismatch: %x", sps)
	}
	if !bytes.Equal(pps, mustHex(t, testPPSHex)) {
		t.Errorf("pps mismatch: %x", pps)
	}
}

func TestIsAnnexBKeyframe(t *testing.T) {
	if !IsAnnexBKeyframe(makeAnnexB(t, true, true)) {
		t.Error("expected IDR to be a keyframe")
	}
	if IsAnnexBKeyframe(makeAnnexB(t, false, false)) {
		t.Error("non-IDR should not be a keyframe")
	}
}

func TestIsAnnexBKeyframeHEVC(t *testing.T) {
	if !IsAnnexBKeyframeHEVC(makeHEVCAnnexB(t, true, true)) {
		t.Error("expected HEVC IDR to be a keyframe")
	}
	if IsAnnexBKeyframeHEVC(makeHEVCAnnexB(t, false, false)) {
		t.Error("HEVC non-IDR should not be a keyframe")
	}
	cra := append([]byte{0, 0, 0, 1, 0x2A, 0x01}, bytes.Repeat([]byte{0x33}, 16)...)
	if IsAnnexBKeyframeHEVC(cra) {
		t.Error("HEVC CRA should not be treated as keyframe for fragment boundaries")
	}
}

func TestNormalizeToAnnexB_FromLengthPrefixedHEVC(t *testing.T) {
	annexB := makeHEVCAnnexB(t, true, true)
	packet := avc.ConvertByteStreamToNaluSample(append([]byte(nil), annexB...))
	got := NormalizeToAnnexB(packet)

	nalus := avc.ExtractNalusFromByteStream(got)
	if len(nalus) != 4 {
		t.Fatalf("normalized nalu count=%d want 4", len(nalus))
	}
	if nt := hevc.GetNaluType(nalus[0][0]); nt != hevc.NALU_VPS {
		t.Fatalf("first nalu type=%d want VPS", nt)
	}
	if nt := hevc.GetNaluType(nalus[3][0]); nt != hevc.NALU_IDR_W_RADL {
		t.Fatalf("last nalu type=%d want IDR", nt)
	}
}

func TestNormalizeToAnnexB_From3ByteLengthPrefixedHEVC(t *testing.T) {
	annexB := makeHEVCAnnexB(t, true, true)
	nalus := avc.ExtractNalusFromByteStream(annexB)
	packet3 := make([]byte, 0, len(annexB))
	for _, nalu := range nalus {
		nLen := len(nalu)
		packet3 = append(packet3, byte(nLen>>16), byte(nLen>>8), byte(nLen))
		packet3 = append(packet3, nalu...)
	}
	got := NormalizeToAnnexB(packet3)
	roundtrip := avc.ExtractNalusFromByteStream(got)
	if len(roundtrip) != len(nalus) {
		t.Fatalf("normalized nalu count=%d want %d", len(roundtrip), len(nalus))
	}
	if nt := hevc.GetNaluType(roundtrip[len(roundtrip)-1][0]); nt != hevc.NALU_IDR_W_RADL {
		t.Fatalf("last nalu type=%d want IDR", nt)
	}
}

// TestRemuxer_RoundTrip writes init+segment, then parses the bytes back
// through mp4ff and checks track + sample counts.
func TestRemuxer_RoundTrip(t *testing.T) {
	sps := mustHex(t, testSPSHex)
	pps := mustHex(t, testPPSHex)

	r := NewRemuxer(90000)

	var buf bytes.Buffer
	if err := r.WriteInit(&buf, [][]byte{sps}, [][]byte{pps}); err != nil {
		t.Fatalf("WriteInit: %v", err)
	}
	samples := []Sample{
		{AnnexB: makeAnnexB(t, true, true), DurationTicks: 3000, IsKeyframe: true},
		{AnnexB: makeAnnexB(t, false, false), DurationTicks: 3000, IsKeyframe: false},
	}
	if err := r.WriteSegment(&buf, samples); err != nil {
		t.Fatalf("WriteSegment: %v", err)
	}

	parsed, err := mp4.DecodeFile(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodeFile: %v", err)
	}
	if !parsed.IsFragmented() {
		t.Fatalf("expected fragmented MP4")
	}
	if parsed.Moov == nil || len(parsed.Moov.Traks) != 1 {
		t.Fatalf("want 1 trak in moov, got %d", len(parsed.Moov.Traks))
	}
	if got := parsed.Moov.Traks[0].Mdia.Hdlr.HandlerType; got != "vide" {
		t.Errorf("handler=%q want vide", got)
	}
	if len(parsed.Segments) != 1 {
		t.Fatalf("want 1 media segment, got %d", len(parsed.Segments))
	}
	frag := parsed.Segments[0].Fragments[0]
	if got := int(frag.Moof.Traf.Trun.SampleCount()); got != 2 {
		t.Errorf("trun samples=%d want 2", got)
	}
	if got := frag.Moof.Traf.Tfdt.BaseMediaDecodeTime(); got != 0 {
		t.Errorf("first tfdt baseMediaDecodeTime=%d want 0", got)
	}
}

func TestRemuxer_RejectsNonKeyframeFirst(t *testing.T) {
	r := NewRemuxer(90000)
	if err := r.WriteInit(&bytes.Buffer{}, [][]byte{mustHex(t, testSPSHex)}, [][]byte{mustHex(t, testPPSHex)}); err != nil {
		t.Fatalf("WriteInit: %v", err)
	}
	err := r.WriteSegment(&bytes.Buffer{}, []Sample{
		{AnnexB: makeAnnexB(t, false, false), DurationTicks: 3000, IsKeyframe: false},
	})
	if err == nil {
		t.Fatal("expected error for non-keyframe-first fragment")
	}
}

func TestRemuxer_HEVCRoundTrip(t *testing.T) {
	vps := mustHex(t, testHEVCVPSHex)
	sps := mustHex(t, testHEVCSPSHex)
	pps := mustHex(t, testHEVCPPSHex)
	r := NewRemuxerForCodec("h265", 90000)
	var buf bytes.Buffer
	if err := r.WriteInitHEVC(&buf, [][]byte{vps}, [][]byte{sps}, [][]byte{pps}); err != nil {
		t.Fatalf("WriteInitHEVC: %v", err)
	}
	samples := []Sample{
		{AnnexB: makeHEVCAnnexB(t, true, true), DurationTicks: 3000, IsKeyframe: true},
		{AnnexB: makeHEVCAnnexB(t, false, false), DurationTicks: 3000, IsKeyframe: false},
	}
	if err := r.WriteSegment(&buf, samples); err != nil {
		t.Fatalf("WriteSegment: %v", err)
	}
	parsed, err := mp4.DecodeFile(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("DecodeFile: %v", err)
	}
	if !parsed.IsFragmented() {
		t.Fatalf("expected fragmented MP4")
	}
	if len(parsed.Segments) == 0 {
		t.Fatalf("expected media segments")
	}
}

func TestDecodeFoxgloveCompressedVideo(t *testing.T) {
	// Build a tiny CompressedVideo proto: only fields 3 (data) and 4 (format).
	// tag for field 3, wire 2 = (3<<3)|2 = 0x1a
	// tag for field 4, wire 2 = (4<<3)|2 = 0x22
	payload := []byte{0xde, 0xad, 0xbe, 0xef}
	var buf bytes.Buffer
	buf.WriteByte(0x1a)
	buf.WriteByte(byte(len(payload)))
	buf.Write(payload)
	buf.WriteByte(0x22)
	buf.WriteByte(4)
	buf.WriteString("h264")

	data, format, err := DecodeFoxgloveCompressedVideo(buf.Bytes())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Errorf("data=%x", data)
	}
	if format != "h264" {
		t.Errorf("format=%q", format)
	}
}
