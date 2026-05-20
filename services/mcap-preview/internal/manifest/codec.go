package manifest

import (
	"io"
	"strings"

	"github.com/foxglove/mcap/go/mcap"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/remux"
)

// DetectTopicCodec scans up to a handful of messages on topic to determine h264/h265.
// rs is rewound to the start on return when possible.
func DetectTopicCodec(rs io.ReadSeeker, topic string) (string, bool) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return "", false
	}
	r, err := mcap.NewReader(rs)
	if err != nil {
		return "", false
	}
	defer r.Close()
	it, err := r.Messages(mcap.WithTopics([]string{topic}))
	if err != nil {
		return "", false
	}
	var msg mcap.Message
	for i := 0; i < 24; i++ {
		_, ch, m, err := it.NextInto(&msg)
		if err != nil || m == nil {
			return "", false
		}
		if ch == nil || ch.Topic != topic {
			continue
		}
		annexB, format, derr := remux.DecodeFoxgloveCompressedVideo(m.Data)
		if derr != nil {
			continue
		}
		if codec, ok := NormalizeCodec(format); ok {
			return codec, true
		}
		annexB = remux.NormalizeToAnnexB(annexB)
		if _, _, _, ok := remux.ExtractVPSPPSHEVC(annexB); ok {
			return "h265", true
		}
		if _, _, ok := remux.ExtractSPSPPS(annexB); ok {
			return "h264", true
		}
	}
	return "", false
}

// NormalizeCodec maps foxglove.CompressedVideo format strings to h264/h265.
func NormalizeCodec(format string) (string, bool) {
	f := strings.ToLower(strings.TrimSpace(format))
	switch f {
	case "h264", "avc", "avc1":
		return "h264", true
	case "h265", "hevc", "hev1", "hvc1":
		return "h265", true
	default:
		return "", false
	}
}
