package server

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/foxglove/mcap/go/mcap"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/gcsrs"
)

// SPS / PPS shared with the remux package tests.
const (
	segTestSPSHex = "67640020accac05005bb0169e0000003002000000c9c4c000432380008647c12401cb1c31380"
	segTestPPSHex = "68b5df20"
	segTestHEVCVPSHex = "40010c01ffff022000000300b0000003000003007b18b024"
	segTestHEVCSPSHex = "420101022000000300b0000003000003007ba0078200887db6718b92448053888892cf24a69272c9124922dc91aa48fca223ff000100016a02020201"
	segTestHEVCPPSHex = "4401c0252f053240"
)

func segHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex: %v", err)
	}
	return b
}

// makeAU mirrors makeAnnexB from remux/h264_fmp4_test.go but is duplicated
// here to keep test helpers package-local.
func makeAU(t *testing.T, withPS, idr bool) []byte {
	t.Helper()
	var b bytes.Buffer
	sc := []byte{0, 0, 0, 1}
	if withPS {
		b.Write(sc)
		b.Write(segHex(t, segTestSPSHex))
		b.Write(sc)
		b.Write(segHex(t, segTestPPSHex))
	}
	b.Write(sc)
	if idr {
		b.WriteByte(0x65)
	} else {
		b.WriteByte(0x41)
	}
	for i := 0; i < 32; i++ {
		b.WriteByte(byte(0x10 + (i & 0x0f)))
	}
	return b.Bytes()
}

func makeHEVCAU(t *testing.T, withPS, idr bool) []byte {
	t.Helper()
	var b bytes.Buffer
	sc := []byte{0, 0, 0, 1}
	if withPS {
		b.Write(sc)
		b.Write(segHex(t, segTestHEVCVPSHex))
		b.Write(sc)
		b.Write(segHex(t, segTestHEVCSPSHex))
		b.Write(sc)
		b.Write(segHex(t, segTestHEVCPPSHex))
	}
	b.Write(sc)
	if idr {
		b.Write([]byte{0x26, 0x01}) // type 19 (IDR_W_RADL)
	} else {
		b.Write([]byte{0x02, 0x01}) // type 1 (TRAIL_R)
	}
	for i := 0; i < 32; i++ {
		b.WriteByte(byte(0x20 + (i & 0x0f)))
	}
	return b.Bytes()
}

// encodeCompressedVideo wraps the access unit + format string in a
// foxglove.CompressedVideo proto wire payload (fields 3 and 4 only).
func encodeCompressedVideo(data []byte, format string) []byte {
	var b bytes.Buffer
	// data: tag = (3<<3)|2 = 0x1a
	b.WriteByte(0x1a)
	b.Write(varint(uint64(len(data))))
	b.Write(data)
	// format: tag = (4<<3)|2 = 0x22
	b.WriteByte(0x22)
	b.Write(varint(uint64(len(format))))
	b.WriteString(format)
	return b.Bytes()
}

func varint(v uint64) []byte {
	var out []byte
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v >>= 7
	}
	out = append(out, byte(v))
	return out
}

// buildVideoMCAP writes a minimal indexed MCAP with one video channel and
// `format` per-message format string. Returns the bytes.
func buildVideoMCAP(t *testing.T, schemaName, format string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := mcap.NewWriter(&buf, &mcap.WriterOptions{
		Chunked:    true,
		ChunkSize:  64,
		IncludeCRC: true,
	})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.WriteHeader(&mcap.Header{Profile: "test", Library: "preview-test"}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if err := w.WriteSchema(&mcap.Schema{ID: 1, Name: schemaName, Encoding: "protobuf", Data: []byte("x")}); err != nil {
		t.Fatalf("WriteSchema: %v", err)
	}
	if err := w.WriteChannel(&mcap.Channel{ID: 1, SchemaID: 1, Topic: "/camera/front", MessageEncoding: "protobuf"}); err != nil {
		t.Fatalf("WriteChannel: %v", err)
	}

	// Two messages: first IDR with SPS/PPS, second non-IDR.
	make := makeAU
	switch strings.ToLower(format) {
	case "hevc", "h265", "hev1", "hvc1":
		make = makeHEVCAU
	}
	au0 := encodeCompressedVideo(make(t, true, true), format)
	au1 := encodeCompressedVideo(make(t, false, false), format)

	if err := w.WriteMessage(&mcap.Message{ChannelID: 1, Sequence: 0, LogTime: 1_000_000_000, Data: au0}); err != nil {
		t.Fatalf("WriteMessage 0: %v", err)
	}
	if err := w.WriteMessage(&mcap.Message{ChannelID: 1, Sequence: 1, LogTime: 1_033_000_000, Data: au1}); err != nil {
		t.Fatalf("WriteMessage 1: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return buf.Bytes()
}

func buildMixedTopicMCAP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := mcap.NewWriter(&buf, &mcap.WriterOptions{
		Chunked:    true,
		ChunkSize:  64,
		IncludeCRC: true,
	})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.WriteHeader(&mcap.Header{Profile: "test", Library: "preview-test"}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if err := w.WriteSchema(&mcap.Schema{ID: 1, Name: "foxglove.CompressedVideo", Encoding: "protobuf", Data: []byte("x")}); err != nil {
		t.Fatalf("WriteSchema video: %v", err)
	}
	if err := w.WriteSchema(&mcap.Schema{ID: 2, Name: "std_msgs/String", Encoding: "protobuf", Data: []byte("x")}); err != nil {
		t.Fatalf("WriteSchema string: %v", err)
	}
	if err := w.WriteChannel(&mcap.Channel{ID: 1, SchemaID: 1, Topic: "/camera/front", MessageEncoding: "protobuf"}); err != nil {
		t.Fatalf("WriteChannel video: %v", err)
	}
	if err := w.WriteChannel(&mcap.Channel{ID: 2, SchemaID: 2, Topic: "/debug/text", MessageEncoding: "protobuf"}); err != nil {
		t.Fatalf("WriteChannel text: %v", err)
	}
	au := encodeCompressedVideo(makeAU(t, true, true), "h264")
	if err := w.WriteMessage(&mcap.Message{ChannelID: 1, Sequence: 0, LogTime: 1_000_000_000, Data: au}); err != nil {
		t.Fatalf("WriteMessage video: %v", err)
	}
	if err := w.WriteMessage(&mcap.Message{ChannelID: 2, Sequence: 0, LogTime: 1_000_001_000, Data: []byte("hello")}); err != nil {
		t.Fatalf("WriteMessage text: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return buf.Bytes()
}

func buildDualCodecMCAP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := mcap.NewWriter(&buf, &mcap.WriterOptions{
		Chunked:    true,
		ChunkSize:  64,
		IncludeCRC: true,
	})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.WriteHeader(&mcap.Header{Profile: "test", Library: "preview-test"}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if err := w.WriteSchema(&mcap.Schema{ID: 1, Name: "foxglove.CompressedVideo", Encoding: "protobuf", Data: []byte("x")}); err != nil {
		t.Fatalf("WriteSchema video: %v", err)
	}
	if err := w.WriteChannel(&mcap.Channel{ID: 1, SchemaID: 1, Topic: "/camera/front", MessageEncoding: "protobuf"}); err != nil {
		t.Fatalf("WriteChannel front: %v", err)
	}
	if err := w.WriteChannel(&mcap.Channel{ID: 2, SchemaID: 1, Topic: "/camera/side", MessageEncoding: "protobuf"}); err != nil {
		t.Fatalf("WriteChannel side: %v", err)
	}
	// front topic is HEVC; side topic is H.264. Auto-pick should prefer H.264.
	hevc := encodeCompressedVideo(makeHEVCAU(t, true, true), "hevc")
	h264 := encodeCompressedVideo(makeAU(t, true, true), "h264")
	if err := w.WriteMessage(&mcap.Message{ChannelID: 1, Sequence: 0, LogTime: 1_000_000_000, Data: hevc}); err != nil {
		t.Fatalf("WriteMessage hevc: %v", err)
	}
	if err := w.WriteMessage(&mcap.Message{ChannelID: 2, Sequence: 0, LogTime: 1_009_000_000, Data: h264}); err != nil {
		t.Fatalf("WriteMessage h264: %v", err)
	}
	h264Follow := encodeCompressedVideo(makeAU(t, false, false), "h264")
	if err := w.WriteMessage(&mcap.Message{ChannelID: 2, Sequence: 1, LogTime: 1_020_000_000, Data: h264Follow}); err != nil {
		t.Fatalf("WriteMessage h264 follow: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return buf.Bytes()
}

func segUpstream(t *testing.T, mcapSize int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/assets/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{
			"asset_id":"abc123",
			"lifecycle_state":"ready",
			"mcap":{"mcap_uri":"gs://b/o.mcap","size_bytes":`+itoa(mcapSize)+`},
			"window":{"start_timestamp_ns":0,"end_timestamp_ns":0,"duration_ms":0}
		}`)
	})
	return httptest.NewServer(mux)
}

func segUpstreamWithWindow(t *testing.T, mcapSize int, startNs, endNs int64) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/assets/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{
			"asset_id":"abc123",
			"lifecycle_state":"ready",
			"mcap":{"mcap_uri":"gs://b/o.mcap","size_bytes":`+itoa(mcapSize)+`},
			"window":{"start_timestamp_ns":`+itoa(int(startNs))+`,"end_timestamp_ns":`+itoa(int(endNs))+`,"duration_ms":0}
		}`)
	})
	return httptest.NewServer(mux)
}

func TestSegment_HappyPath(t *testing.T) {
	mcapBytes := buildVideoMCAP(t, "foxglove.CompressedVideo", "h264")
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "video/mp4" {
		t.Errorf("content-type=%q", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("cache-control=%q", got)
	}
	body := w.Body.Bytes()
	if !bytes.Contains(body[:64], []byte("ftyp")) {
		t.Fatalf("expected ftyp early in body, got %s", hex.EncodeToString(body[:32]))
	}
	parsed, err := mp4.DecodeFile(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("DecodeFile: %v", err)
	}
	if !parsed.IsFragmented() {
		t.Fatalf("expected fragmented MP4")
	}
	if len(parsed.Segments) == 0 {
		t.Fatalf("no media segments emitted")
	}
}

func TestSegment_RejectsNonH264(t *testing.T) {
	// Schema is CompressedVideo but per-message format is unsupported.
	mcapBytes := buildVideoMCAP(t, "foxglove.CompressedVideo", "av1")
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL: upstream.URL,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"UNSUPPORTED_PREVIEW_CODEC"`) {
		t.Errorf("body=%s", w.Body.String())
	}
}

func TestSegment_HEVCHappyPath(t *testing.T) {
	mcapBytes := buildVideoMCAP(t, "foxglove.CompressedVideo", "hevc")
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Preview-Codec"); got != "h264" {
		t.Fatalf("codec=%q want h264", got)
	}
	if got := w.Header().Get("X-Preview-Codec-Source"); got != "h265-transcoded-stream" && got != "h265-transcoded" && got != "h265-transcoded-streaming" {
		t.Fatalf("codec-source=%q want h265-transcoded(-stream(ing))", got)
	}
}

func TestSegment_PreferH264TopicWhenAvailable(t *testing.T) {
	mcapBytes := buildDualCodecMCAP(t)
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Preview-Topic"); got != "/camera/side" {
		t.Fatalf("topic=%q want /camera/side", got)
	}
	if got := w.Header().Get("X-Preview-Codec"); got != "h264" {
		t.Fatalf("codec=%q want h264", got)
	}
}

func TestSegment_H265HintFallsBackToH264Topic(t *testing.T) {
	mcapBytes := buildDualCodecMCAP(t)
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4?topic=/camera/front", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Preview-Topic"); got != "/camera/side" {
		t.Fatalf("topic=%q want /camera/side", got)
	}
	if got := w.Header().Get("X-Preview-Codec"); got != "h264" {
		t.Fatalf("codec=%q want h264", got)
	}
}

func TestSegment_RejectsNonVideoSchema(t *testing.T) {
	// pickTopic should fail with no candidate channel.
	mcapBytes := buildVideoMCAP(t, "std_msgs/String", "h264")
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL: upstream.URL,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"NO_PREVIEW_TOPIC"`) {
		t.Errorf("body=%s", w.Body.String())
	}
}

func TestSegment_InvalidTopicHint_Returns422(t *testing.T) {
	mcapBytes := buildVideoMCAP(t, "foxglove.CompressedVideo", "h264")
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL: upstream.URL,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4?topic=/camera/not-exist", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"NO_PREVIEW_TOPIC"`) {
		t.Fatalf("body=%s", w.Body.String())
	}
}

func TestSegment_NonVideoTopicHint_Returns422(t *testing.T) {
	mcapBytes := buildMixedTopicMCAP(t)
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL: upstream.URL,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4?topic=/debug/text", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"NO_PREVIEW_TOPIC"`) {
		t.Fatalf("body=%s", w.Body.String())
	}
}

func TestSegment_TokenViaQueryParam(t *testing.T) {
	mcapBytes := buildVideoMCAP(t, "foxglove.CompressedVideo", "h264")
	upstream := segUpstream(t, len(mcapBytes))
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/preview/assets/abc123/segment.mp4?grace_token=tok", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSegment_NormalizesMillisecondWindow(t *testing.T) {
	mcapBytes := buildVideoMCAP(t, "foxglove.CompressedVideo", "h264")
	// Message log times are around 1_000_000_000ns and 1_033_000_000ns, so this
	// window intentionally uses milliseconds to validate unit normalization.
	upstream := segUpstreamWithWindow(t, len(mcapBytes), 1000, 1100)
	defer upstream.Close()

	r := New(Config{
		UpstreamBaseURL:       upstream.URL,
		GraceTokenPassthrough: true,
		OpenMCAP: func(_ context.Context, _ string) (io.ReadSeeker, *gcsrs.Stats, func(), error) {
			return bytes.NewReader(mcapBytes), nil, nil, nil
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4", nil)
	req.Header.Set("X-Grace-Token", "tok")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if w.Body.Len() == 0 {
		t.Fatalf("expected non-empty mp4 body")
	}
}

func TestSegment_MissingTokenIs401(t *testing.T) {
	r := New(Config{UpstreamBaseURL: "http://unused"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/assets/abc123/segment.mp4", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestMonotonicDelta(t *testing.T) {
	tests := []struct {
		name      string
		last      uint64
		current   uint64
		wantDelta uint64
		wantLast  uint64
	}{
		{name: "normal increasing", last: 100, current: 140, wantDelta: 40, wantLast: 140},
		{name: "equal timestamp", last: 100, current: 100, wantDelta: 0, wantLast: 100},
		{name: "out of order timestamp", last: 100, current: 90, wantDelta: 0, wantLast: 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotDelta, gotLast := monotonicDelta(tc.last, tc.current)
			if gotDelta != tc.wantDelta || gotLast != tc.wantLast {
				t.Fatalf("delta=%d last=%d, want delta=%d last=%d", gotDelta, gotLast, tc.wantDelta, tc.wantLast)
			}
		})
	}
}

func TestShouldForceFlushPending(t *testing.T) {
	tests := []struct {
		name       string
		samples    int
		durTicks   uint64
		bytes      uint64
		wantForced bool
	}{
		{name: "below all guards", samples: maxPendingSamples - 1, durTicks: maxPendingDurTicks - 1, bytes: maxPendingBytes - 1, wantForced: false},
		{name: "sample guard", samples: maxPendingSamples, durTicks: 0, bytes: 0, wantForced: true},
		{name: "duration guard", samples: 1, durTicks: maxPendingDurTicks, bytes: 0, wantForced: true},
		{name: "byte guard", samples: 1, durTicks: 1, bytes: maxPendingBytes, wantForced: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldForceFlushPending(tc.samples, tc.durTicks, tc.bytes)
			if got != tc.wantForced {
				t.Fatalf("forced=%v want=%v", got, tc.wantForced)
			}
		})
	}
}
