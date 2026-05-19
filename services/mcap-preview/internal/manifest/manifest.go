// Package manifest builds the preview manifest response by reading just the
// MCAP summary section (Info) — no message scan. Topic enumeration and
// per-window chunk listing therefore cost O(channels + chunk_indexes), which
// is what makes the page-cached GCS reader practical.
package manifest

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/foxglove/mcap/go/mcap"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/gcsrs"
)

// Locator carries the upstream mcap-locator fields the manifest needs.
//
// We accept just the bits we use rather than re-defining the full upstream
// response shape; the caller is responsible for fetching+decoding it.
type Locator struct {
	AssetID          string
	McapURI          string
	SizeBytes        int64
	StartTimestampNs int64
	EndTimestampNs   int64
	DurationMs       int64
}

// Response is the JSON body returned by GET /api/v1/preview/assets/:id/manifest.
type Response struct {
	AssetID              string                 `json:"asset_id"`
	Mcap                 McapRef                `json:"mcap"`
	Window               Window                 `json:"window"`
	CandidateVideoTopics []CandidateVideoTopic  `json:"candidate_video_topics"`
	Sources              []PreviewSource        `json:"sources"`
	RecommendedSourceID  string                 `json:"recommended_source_id,omitempty"`
	ChunksInWindow       []ChunkInWindow        `json:"chunks_in_window"`
	Stats                map[string]interface{} `json:"stats"`
}

type McapRef struct {
	McapURI   string `json:"mcap_uri"`
	SizeBytes int64  `json:"size_bytes"`
}

type Window struct {
	StartTimestampNs int64 `json:"start_timestamp_ns"`
	EndTimestampNs   int64 `json:"end_timestamp_ns"`
	DurationMs       int64 `json:"duration_ms"`
}

type CandidateVideoTopic struct {
	Topic              string `json:"topic"`
	SchemaName         string `json:"schema_name"`
	Encoding           string `json:"encoding"`
	MessageCountWindow uint64 `json:"message_count_in_window"`
}

type PreviewSource struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Codec string `json:"codec,omitempty"`
	Topic string `json:"topic,omitempty"`
	URL   string `json:"url"`
}

type ChunkInWindow struct {
	ChunkIndex       int    `json:"chunk_index"`
	MessageStartTime uint64 `json:"message_start_time"`
	MessageEndTime   uint64 `json:"message_end_time"`
	ChunkStartOffset uint64 `json:"chunk_start_offset"`
	ChunkLength      uint64 `json:"chunk_length"`
}

// Build reads MCAP summary/index from rs and assembles the manifest.
//
// stats is optional; if non-nil its counters are surfaced under "stats" so
// callers can audit GCS IO cost. Window comes from Locator (asset row), not
// from the MCAP file itself — assets may map onto a sub-range of the MCAP.
func Build(rs io.ReadSeeker, loc Locator, stats *gcsrs.Stats) (*Response, error) {
	r, err := mcap.NewReader(rs)
	if err != nil {
		return nil, fmt.Errorf("mcap.NewReader: %w", err)
	}
	defer r.Close()

	info, err := r.Info()
	if err != nil {
		return nil, fmt.Errorf("mcap Info: %w", err)
	}

	wStart, wEnd, windowMode := NormalizeWindow(loc.StartTimestampNs, loc.EndTimestampNs, info)

	resp := &Response{
		AssetID: loc.AssetID,
		Mcap: McapRef{
			McapURI:   loc.McapURI,
			SizeBytes: loc.SizeBytes,
		},
		Window: Window{
			StartTimestampNs: loc.StartTimestampNs,
			EndTimestampNs:   loc.EndTimestampNs,
			DurationMs:       loc.DurationMs,
		},
		CandidateVideoTopics: candidateVideoTopics(info),
		ChunksInWindow:       chunksInWindow(info, wStart, wEnd),
		Stats:                map[string]interface{}{},
	}
	resp.Sources, resp.RecommendedSourceID = buildPreviewSources(loc.AssetID, resp.CandidateVideoTopics)
	if len(resp.ChunksInWindow) == 0 && info.Statistics != nil &&
		(info.Statistics.MessageStartTime != 0 || info.Statistics.MessageEndTime != 0) {
		// Empty intersection is usually a unit mismatch from upstream
		// locator metadata; prefer returning useful preview chunks.
		resp.ChunksInWindow = chunksInWindow(info, info.Statistics.MessageStartTime, info.Statistics.MessageEndTime)
		if len(resp.ChunksInWindow) > 0 {
			windowMode = windowMode + "_empty_to_stats"
			wStart = info.Statistics.MessageStartTime
			wEnd = info.Statistics.MessageEndTime
		}
	}

	if stats != nil {
		resp.Stats["gcs_range_requests"] = stats.RangeCalls
		resp.Stats["gcs_bytes_read"] = stats.BytesRead
	}
	resp.Stats["window_mode"] = windowMode
	resp.Stats["window_effective_start_ns"] = wStart
	resp.Stats["window_effective_end_ns"] = wEnd
	return resp, nil
}

func buildPreviewSources(assetID string, topics []CandidateVideoTopic) ([]PreviewSource, string) {
	sources := make([]PreviewSource, 0, len(topics))
	bestID := ""
	bestScore := -1
	for i, t := range topics {
		if !isSupportedVideoSchema(t.SchemaName) {
			continue
		}
		if strings.Contains(strings.ToLower(t.Topic), "timestamp_map") {
			continue
		}
		// Only live segment sources are emitted here; file-based preview_mp4
		// is asset-domain metadata and should be merged by API gateway/UI layer.
		base := fmt.Sprintf("/api/v1/preview/assets/%s/segment.mp4", url.PathEscape(assetID))
		topicEscaped := url.QueryEscape(t.Topic)
		sourceID := fmt.Sprintf("live_topic_%d", i)
		sources = append(sources, PreviewSource{
			ID:    sourceID,
			Kind:  "live",
			Topic: t.Topic,
			URL:   fmt.Sprintf("%s?topic=%s", base, topicEscaped),
		})
		score := topicPreferenceScore(t.Topic)
		if score > bestScore {
			bestScore = score
			bestID = sourceID
		}
	}
	return sources, bestID
}

func isSupportedVideoSchema(schemaName string) bool {
	return strings.Contains(strings.ToLower(schemaName), "compressedvideo")
}

func topicPreferenceScore(topic string) int {
	t := strings.ToLower(topic)
	switch {
	case strings.Contains(t, "front"):
		return 40
	case strings.Contains(t, "main"):
		return 30
	case strings.Contains(t, "side"):
		return 20
	case strings.Contains(t, "down"), strings.Contains(t, "bottom"):
		return 10
	default:
		return 0
	}
}

func candidateVideoTopics(info *mcap.Info) []CandidateVideoTopic {
	out := make([]CandidateVideoTopic, 0, len(info.Channels))
	for id, ch := range info.Channels {
		schemaName := ""
		if s := info.Schemas[ch.SchemaID]; s != nil {
			schemaName = s.Name
		}
		if !looksLikeVideoOrImage(schemaName) {
			continue
		}
		// Message count in window requires a message scan, which the
		// skeleton intentionally avoids; report 0 for now.
		_ = id
		out = append(out, CandidateVideoTopic{
			Topic:              ch.Topic,
			SchemaName:         schemaName,
			Encoding:           ch.MessageEncoding,
			MessageCountWindow: 0,
		})
	}
	return out
}

func looksLikeVideoOrImage(schemaName string) bool {
	if schemaName == "" {
		return false
	}
	n := strings.ToLower(schemaName)
	return strings.Contains(n, "video") ||
		strings.Contains(n, "image") ||
		strings.Contains(n, "h264") ||
		strings.Contains(n, "h265")
}

func chunksInWindow(info *mcap.Info, wStart, wEnd uint64) []ChunkInWindow {
	out := make([]ChunkInWindow, 0, len(info.ChunkIndexes))
	for i, ci := range info.ChunkIndexes {
		if ci == nil {
			continue
		}
		// Skip chunks that strictly precede or follow the window. When
		// the window is zero-length we still keep chunks whose range
		// contains it, which is the natural set semantic.
		if ci.MessageEndTime < wStart || ci.MessageStartTime > wEnd {
			continue
		}
		out = append(out, ChunkInWindow{
			ChunkIndex:       i,
			MessageStartTime: ci.MessageStartTime,
			MessageEndTime:   ci.MessageEndTime,
			ChunkStartOffset: ci.ChunkStartOffset,
			ChunkLength:      ci.ChunkLength,
		})
	}
	return out
}
