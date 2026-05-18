package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"os"
	"path/filepath"
	"strconv"
	"sort"
	"strings"
	"time"

	"github.com/foxglove/mcap/go/mcap"
	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/manifest"
	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/remux"
)

// Additional error code reserved for the segment endpoint.
const (
	codeUnsupportedPreviewCodec = "UNSUPPORTED_PREVIEW_CODEC"
	codePreviewNoSPSPPS         = "PREVIEW_NO_PARAMETER_SETS"
	codeNoCandidateTopic        = "NO_PREVIEW_TOPIC"
	codeWindowTimebaseMismatch  = "WINDOW_TIMEBASE_MISMATCH"
)

// videoTimescale is the fMP4 media timescale we emit. 90 kHz is the standard
// choice for H.264 in MPEG transports — high enough to express common frame
// intervals (3000 ticks @ 30 fps, 3750 @ 24 fps) without rounding drift.
const videoTimescale uint32 = 90000

// targetFragmentDurTicks is roughly one second's worth of video. We cut a
// fragment whenever we have at least this much accumulated and we are about
// to receive a new keyframe. The exact size doesn't matter for correctness;
// ~1s keeps init-to-first-paint short while limiting moof overhead.
const targetFragmentDurTicks uint64 = 90000

// minFirstFragmentDurTicks asks remux to accumulate a larger opening
// fragment before the first cut, reducing startup stalls on long-GOP streams.
const minFirstFragmentDurTicks uint64 = 270000

// Guardrail for long-GOP / missing-keyframe streams to avoid unbounded
// pending-sample accumulation before fragment cuts.
const (
	maxPendingSamples    = 240
	maxPendingDurTicks   = 900000 // ~10s at 90kHz
	maxPendingBytes      = 32 << 20
	fallbackFrameDurTick = 3000   // ~30fps at 90kHz
	maxClipSeconds       = 30
)

var previewMP4Cache = newPreviewCache("/tmp/mcap-preview-cache", 30*time.Minute)

// segmentHandler handles GET /api/v1/preview/assets/:id/segment.mp4.
//
// It re-uses fetchLocator from the manifest handler; the same upstream call
// shape applies (X-Grace-Token forwarded, 4xx/5xx propagated). For media
// elements we additionally accept ?grace_token= because <video> tags cannot
// send custom headers.
func segmentHandler(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		assetID := c.Param("id")
		if !assetIDPattern.MatchString(assetID) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid asset id", nil)
			return
		}
		token := extractGraceToken(c)
		if token == "" {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "missing X-Grace-Token (or ?grace_token=, grace_session cookie)")
			return
		}
		if cfg.UpstreamBaseURL == "" || cfg.OpenMCAP == nil {
			httpresp.Unavailable(c, httpresp.CodeServiceUnavailable, "preview not fully configured")
			return
		}

		loc, status, err := fetchLocator(c, cfg, assetID, token)
		if err != nil {
			if status == 0 {
				httpresp.Error(c, http.StatusBadGateway, httpresp.CodeUpstreamError, err.Error(), nil)
			}
			return
		}

		rs, _, closer, err := cfg.acquireMCAP(c.Request.Context(), loc.Mcap.McapURI)
		if err != nil {
			httpresp.Error(c, http.StatusBadGateway, httpresp.CodeUpstreamError, fmt.Sprintf("open mcap: %v", err), nil)
			return
		}
		if closer != nil {
			defer closer()
		}

		topic, err := pickTopic(rs, c.Query("topic"))
		if err != nil {
			httpresp.Error(c, http.StatusUnprocessableEntity, codeNoCandidateTopic, err.Error(), nil)
			return
		}

		startNs, endNs := effectiveRequestedWindowNs(
			loc.Window.StartTimestampNs,
			loc.Window.EndTimestampNs,
			c.Query("start_ns"),
			c.Query("end_ns"),
			c.Query("start_sec"),
			c.Query("end_sec"),
		)

		streamSegments(
			c,
			rs,
			topic,
			startNs,
			endNs,
			loc.Window.DurationMs,
			cfg.StrictWindowValidation,
			effectiveClipSeconds(c.Query("clip_sec")),
			func(ctx context.Context) (io.ReadSeeker, func(), error) {
				rs2, _, closer2, err := cfg.acquireMCAP(ctx, loc.Mcap.McapURI)
				if err != nil {
					return nil, nil, err
				}
				return rs2, closer2, nil
			},
		)
	}
}

func effectiveRequestedWindowNs(
	defaultStart int64,
	defaultEnd int64,
	startNsQ string,
	endNsQ string,
	startSecQ string,
	endSecQ string,
) (int64, int64) {
	start := effectiveWindowStartNs(defaultStart, startNsQ)
	end := effectiveWindowEndNs(defaultEnd, endNsQ)
	// start_sec/end_sec are relative to locator window start; use them only
	// when provided and valid. They intentionally override ns query params.
	if sec, ok := parseNonNegativeFloatSeconds(startSecQ); ok {
		start = defaultStart + int64(sec*1_000_000_000)
	}
	if sec, ok := parseNonNegativeFloatSeconds(endSecQ); ok {
		end = defaultStart + int64(sec*1_000_000_000)
	}
	if end > 0 && start > 0 && end <= start {
		// keep a non-empty window; fall back to default asset window.
		return defaultStart, defaultEnd
	}
	if start < defaultStart {
		start = defaultStart
	}
	if end > 0 && defaultEnd > 0 && end > defaultEnd {
		end = defaultEnd
	}
	return start, end
}

func parseNonNegativeFloatSeconds(q string) (float64, bool) {
	s := strings.TrimSpace(q)
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func effectiveClipSeconds(q string) int64 {
	if strings.TrimSpace(q) == "" {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(q), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	if n > maxClipSeconds {
		return maxClipSeconds
	}
	return n
}

func effectiveWindowStartNs(defaultStart int64, q string) int64 {
	if q == "" {
		return defaultStart
	}
	n, err := strconv.ParseInt(strings.TrimSpace(q), 10, 64)
	if err != nil {
		return defaultStart
	}
	if n > defaultStart {
		return n
	}
	return defaultStart
}

func effectiveWindowEndNs(defaultEnd int64, q string) int64 {
	if q == "" {
		return defaultEnd
	}
	n, err := strconv.ParseInt(strings.TrimSpace(q), 10, 64)
	if err != nil {
		return defaultEnd
	}
	if n > 0 && n < defaultEnd {
		return n
	}
	return defaultEnd
}

// pickTopic returns the topic to stream. If hint is non-empty it must match a
// channel in this MCAP and that channel must be a supported video schema.
// Without hint we choose the best CompressedVideo topic by score.
func pickTopic(rs io.ReadSeeker, hint string) (string, error) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek mcap: %w", err)
	}
	r, err := mcap.NewReader(rs)
	if err != nil {
		return "", fmt.Errorf("mcap.NewReader: %w", err)
	}
	defer r.Close()
	info, err := r.Info()
	if err != nil {
		return "", fmt.Errorf("mcap Info: %w", err)
	}
	if hint = strings.TrimSpace(hint); hint != "" {
		foundHint := false
		for _, ch := range info.Channels {
			if ch == nil || ch.Topic != hint {
				continue
			}
			foundHint = true
			s := info.Schemas[ch.SchemaID]
			if s == nil {
				return "", fmt.Errorf("requested topic %q has no schema", hint)
			}
			if !strings.Contains(strings.ToLower(s.Name), "compressedvideo") {
				return "", fmt.Errorf("requested topic %q is not a supported video schema", hint)
			}
			if codec, ok := detectTopicCodec(rs, hint); ok && codec == "h264" {
				return hint, nil
			}
			break
		}
		if !foundHint {
			return "", fmt.Errorf("requested topic %q not found in mcap", hint)
		}
		// Hint exists but may be unsupported codec (for example h265).
		// Fall through to global H264 selection so preview still works.
	}
	ids := make([]uint16, 0, len(info.Channels))
	for id := range info.Channels {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	bestTopic := ""
	bestScore := -1
	bestH264Topic := ""
	bestH264Score := -1
	for _, id := range ids {
		ch := info.Channels[id]
		if ch == nil {
			continue
		}
		s := info.Schemas[ch.SchemaID]
		if s == nil {
			continue
		}
		n := strings.ToLower(s.Name)
		if !strings.Contains(n, "compressedvideo") {
			continue
		}
		score := topicPreferenceScore(ch.Topic)
		if codec, ok := detectTopicCodec(rs, ch.Topic); ok && codec == "h264" {
			if score > bestH264Score || (score == bestH264Score && (bestH264Topic == "" || ch.Topic < bestH264Topic)) {
				bestH264Topic = ch.Topic
				bestH264Score = score
			}
		}
		if score > bestScore || (score == bestScore && (bestTopic == "" || ch.Topic < bestTopic)) {
			bestTopic = ch.Topic
			bestScore = score
		}
	}
	if bestH264Topic != "" {
		return bestH264Topic, nil
	}
	if bestTopic != "" {
		return bestTopic, nil
	}
	return "", errors.New("no foxglove.CompressedVideo channel found")
}

func detectTopicCodec(rs io.ReadSeeker, topic string) (string, bool) {
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
		if codec, ok := normalizeCodec(format); ok {
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

// streamSegments iterates messages on `topic` within the asset window, remuxes
// to fMP4, and writes init+fragments to the response, flushing after each.
func streamSegments(
	c *gin.Context,
	rs io.ReadSeeker,
	topic string,
	rawStartTs,
	rawEndTs,
	rawDurationMs int64,
	strictWindowValidation bool,
	clipSeconds int64,
	openForTranscode func(ctx context.Context) (io.ReadSeeker, func(), error),
) {
	assetID := c.Param("id")
	// Open a fresh mcap reader — pickTopic already consumed the previous one.
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		httpresp.Internal(c, fmt.Sprintf("seek mcap: %v", err))
		return
	}
	r, err := mcap.NewReader(rs)
	if err != nil {
		httpresp.Internal(c, fmt.Sprintf("mcap reader: %v", err))
		return
	}
	defer r.Close()

	info, err := r.Info()
	if err != nil {
		httpresp.Internal(c, fmt.Sprintf("mcap info: %v", err))
		return
	}
	startNs, endNs, mode := manifest.NormalizeWindow(rawStartTs, rawEndTs, info)
	if rawDurationMs > 0 && endNs > startNs && info != nil && info.Statistics != nil {
		expectedSpanNs := uint64(rawDurationMs) * 1_000_000
		actualSpanNs := endNs - startNs
		if expectedSpanNs > 0 && actualSpanNs > expectedSpanNs*4 {
			// Span far exceeds the declared asset duration → locator
			// units don't match MCAP log_time. Treat rawStart as a
			// nanosecond offset from stats.MessageStartTime so two
			// assets sharing one MCAP get distinct sub-ranges instead
			// of both anchoring at the MCAP head.
			base := info.Statistics.MessageStartTime
			if rawStartTs > 0 {
				startNs = base + uint64(rawStartTs)
			} else {
				startNs = base
			}
			endNs = startNs + expectedSpanNs
			if endNs > info.Statistics.MessageEndTime {
				endNs = info.Statistics.MessageEndTime
			}
			mode = mode + "_span_clamped_offset"
		}
	}
	log.Printf(
		"[preview] asset=%s topic=%s raw_start=%d raw_end=%d raw_duration_ms=%d normalized_start=%d normalized_end=%d mode=%s",
		assetID, topic, rawStartTs, rawEndTs, rawDurationMs, startNs, endNs, mode,
	)
	if strictWindowValidation && strings.Contains(mode, "stats_fallback_mismatch") {
		httpresp.Error(
			c,
			http.StatusUnprocessableEntity,
			codeWindowTimebaseMismatch,
			"asset window timebase mismatches MCAP log_time; preview rejected by strict validation",
			map[string]any{
				"raw_start_timestamp_ns": rawStartTs,
				"raw_end_timestamp_ns":   rawEndTs,
				"normalized_start_ns":    startNs,
				"normalized_end_ns":      endNs,
				"mode":                   mode,
			},
		)
		return
	}
	if strings.Contains(mode, "stats_fallback_mismatch") && !strings.HasSuffix(mode, "_offset") && rawDurationMs > 0 && info.Statistics != nil {
		// Upstream asset windows are occasionally emitted in a different
		// clock domain than MCAP log_time (relative clock vs epoch clock),
		// which would otherwise force us to remux the whole file on each
		// preview request. Derive offset from locator window first so assets
		// sharing one MCAP still map to distinct sub-ranges.
		base := info.Statistics.MessageStartTime
		if rawStartTs > 0 {
			startNs = base + uint64(rawStartTs)
			mode = mode + "_offset"
		} else {
			startNs = base
		}
		spanNs := uint64(rawDurationMs) * 1_000_000
		if rawEndTs > rawStartTs {
			spanNs = uint64(rawEndTs - rawStartTs)
		}
		endNs = startNs + spanNs
		if endNs > info.Statistics.MessageEndTime {
			endNs = info.Statistics.MessageEndTime
		}
	}
	isH265 := false
	if codec, ok := detectTopicCodec(rs, topic); ok && codec == "h265" {
		isH265 = true
	}
	if clipSeconds > 0 {
		clipNs := uint64(clipSeconds) * 1_000_000_000
		if endNs > startNs && endNs-startNs > clipNs {
			endNs = startNs + clipNs
			mode = mode + "_clip"
		}
	}
	if isH265 {
		streamSegmentsViaFFmpeg(c, topic, startNs, endNs, mode, clipSeconds, openForTranscode)
		return
	}
	c.Writer.Header().Set("X-Preview-Topic", topic)
	c.Writer.Header().Set("X-Preview-Window-StartNs", strconv.FormatUint(startNs, 10))
	c.Writer.Header().Set("X-Preview-Window-EndNs", strconv.FormatUint(endNs, 10))
	c.Writer.Header().Set("X-Preview-Mode", mode)
	c.Writer.Header().Set("X-Preview-Clip-Seconds", strconv.FormatInt(clipSeconds, 10))

	opts := []mcap.ReadOpt{mcap.WithTopics([]string{topic})}
	if startNs > 0 {
		opts = append(opts, mcap.AfterNanos(startNs))
	}
	if endNs > 0 {
		opts = append(opts, mcap.BeforeNanos(endNs))
	}
	it, err := r.Messages(opts...)
	if err != nil {
		httpresp.Internal(c, fmt.Sprintf("mcap messages: %v", err))
		return
	}

	muxer := remux.NewRemuxer(videoTimescale)
	flusher, _ := c.Writer.(http.Flusher)

	// First-pass state: we don't write headers until we know the codec is
	// h264 and we have SPS/PPS, otherwise we'd commit to a 200 we cannot
	// later turn into a 415/422.
	var headerWritten bool

	// pending holds samples for the current fragment-in-progress. We flush
	// whenever we see the *next* keyframe (so each fragment starts on IDR
	// and has at least targetFragmentDurTicks of media), or when the
	// stream ends.
	var (
		pending         []remux.Sample
		pendingTotalDur uint64
		pendingBytes    uint64
		totalSamples    uint64
		fragmentCount   uint64
		bytesWritten    uint64
		firstCutDone    bool
		// last log time of the most recently buffered sample, used to
		// derive the previous sample's duration when the next message
		// arrives.
		lastLogTimeNs uint64
		haveLast      bool
		activeCodec   string
	)

	finalize := func(reason string) bool {
		if len(pending) > 0 {
			if err := muxer.WriteSegment(c.Writer, pending); err != nil {
				log.Printf(
					"[preview] asset=%s topic=%s segment_write_error reason=%s samples=%d fragments=%d err=%v",
					assetID, topic, reason, totalSamples, fragmentCount, err,
				)
				return false
			}
			fragmentCount++
			log.Printf(
				"[preview] asset=%s topic=%s fragment_flush reason=%s idx=%d fragment_samples=%d total_samples=%d dur_ticks=%d pending_bytes=%d",
				assetID, topic, reason, fragmentCount, len(pending), totalSamples, pendingTotalDur, pendingBytes,
			)
			pending = pending[:0]
			pendingTotalDur = 0
			pendingBytes = 0
			if flusher != nil {
				flusher.Flush()
			}
		}
		return true
	}

	ctx := c.Request.Context()
	var msg mcap.Message
	for {
		if err := ctx.Err(); err != nil {
			// Client disconnected mid-stream — drop quietly. We've
			// already committed to 200 if headerWritten, so there's
			// nothing useful to write back.
			log.Printf(
				"[preview] asset=%s topic=%s client_done reason=%v samples=%d fragments=%d bytes=%d",
				assetID, topic, err, totalSamples, fragmentCount, bytesWritten,
			)
			return
		}
		_, ch, m, err := it.NextInto(&msg)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if !headerWritten {
				httpresp.Internal(c, fmt.Sprintf("mcap iter: %v", err))
				return
			}
			_ = finalize("iter_error")
			log.Printf(
				"[preview] asset=%s topic=%s iter_error err=%v samples=%d fragments=%d bytes=%d",
				assetID, topic, err, totalSamples, fragmentCount, bytesWritten,
			)
			return
		}
		if m == nil {
			break
		}
		if ch == nil || ch.Topic != topic {
			continue
		}

		annexB, format, derr := remux.DecodeFoxgloveCompressedVideo(m.Data)
		if derr != nil {
			if !headerWritten {
				httpresp.Error(c, http.StatusUnprocessableEntity,
					httpresp.CodeInvalidArgument,
					fmt.Sprintf("decode CompressedVideo: %v", derr), nil)
				return
			}
			continue
		}
		codec, ok := normalizeCodec(format)
		if !ok && strings.TrimSpace(format) != "" {
			if !headerWritten {
				httpresp.Error(c, http.StatusUnsupportedMediaType,
					codeUnsupportedPreviewCodec,
					fmt.Sprintf("only h264/h265 are supported (got %q)", format),
					map[string]any{"format": format})
				return
			}
			continue
		}
		annexB = remux.NormalizeToAnnexB(annexB)
		isKey := remux.IsAnnexBKeyframe(annexB)
		if codec == "h265" {
			isKey = remux.IsAnnexBKeyframeHEVC(annexB)
		}

		if !headerWritten {
			muxer = remux.NewRemuxerForCodec(codec, videoTimescale)
			// Need SPS+PPS to build moov. Foxglove streams typically
			// carry them in-band, prepended to each IDR. If missing
			// we cannot remux.
			if !isKey {
				// Skip non-keyframe samples until we find an IDR.
				continue
			}
			// If upstream did not specify format, auto-detect by probing VPS/SPS/PPS vs SPS/PPS.
			// Mis-labeling h265 as h264 causes a hard decode failure in browser video pipelines.
			if strings.TrimSpace(format) == "" {
				if vps, sps, pps, ok := remux.ExtractVPSPPSHEVC(annexB); ok {
					codec = "h265"
					muxer = remux.NewRemuxerForCodec(codec, videoTimescale)
					activeCodec = codec
					log.Printf("[preview] asset=%s topic=%s detected_codec=%s format=%q", assetID, topic, codec, format)
					var initBuf bytes.Buffer
					if err := muxer.WriteInitHEVC(&initBuf, [][]byte{vps}, [][]byte{sps}, [][]byte{pps}); err != nil {
						httpresp.Internal(c, fmt.Sprintf("write init segment: %v", err))
						return
					}
					c.Writer.Header().Set("Content-Type", "video/mp4")
					c.Writer.Header().Set("Cache-Control", "no-store")
					c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
					c.Writer.Header().Set("X-Accel-Buffering", "no")
					c.Writer.Header().Set("X-Preview-Codec", codec)
					c.Writer.WriteHeader(http.StatusOK)
					w, writeErr := c.Writer.Write(initBuf.Bytes())
					bytesWritten += uint64(w)
					if writeErr != nil {
						log.Printf("[preview] asset=%s topic=%s init_write_error bytes=%d err=%v", assetID, topic, bytesWritten, writeErr)
						return
					}
					if flusher != nil {
						flusher.Flush()
					}
					headerWritten = true
					lastLogTimeNs = m.LogTime
					haveLast = true
					pending = append(pending, remux.Sample{
						AnnexB:        annexB,
						DurationTicks: 0,
						IsKeyframe:    true,
					})
					pendingBytes += uint64(len(annexB))
					totalSamples++
					continue
				}
				// Fall back to h264 if SPS/PPS can be extracted.
				if sps, pps, ok := remux.ExtractSPSPPS(annexB); ok {
					codec = "h264"
					muxer = remux.NewRemuxerForCodec(codec, videoTimescale)
					activeCodec = codec
					log.Printf("[preview] asset=%s topic=%s detected_codec=%s format=%q", assetID, topic, codec, format)
					var initBuf bytes.Buffer
					if err := muxer.WriteInit(&initBuf, [][]byte{sps}, [][]byte{pps}); err != nil {
						httpresp.Internal(c, fmt.Sprintf("write init segment: %v", err))
						return
					}
					c.Writer.Header().Set("Content-Type", "video/mp4")
					c.Writer.Header().Set("Cache-Control", "no-store")
					c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
					c.Writer.Header().Set("X-Accel-Buffering", "no")
					c.Writer.Header().Set("X-Preview-Codec", codec)
					c.Writer.WriteHeader(http.StatusOK)
					w, writeErr := c.Writer.Write(initBuf.Bytes())
					bytesWritten += uint64(w)
					if writeErr != nil {
						log.Printf("[preview] asset=%s topic=%s init_write_error bytes=%d err=%v", assetID, topic, bytesWritten, writeErr)
						return
					}
					if flusher != nil {
						flusher.Flush()
					}
					headerWritten = true
					lastLogTimeNs = m.LogTime
					haveLast = true
					pending = append(pending, remux.Sample{
						AnnexB:        annexB,
						DurationTicks: 0,
						IsKeyframe:    true,
					})
					pendingBytes += uint64(len(annexB))
					totalSamples++
					continue
				}
				httpresp.Error(c, http.StatusUnprocessableEntity,
					codePreviewNoSPSPPS,
					"first keyframe sample lacks parameter sets; cannot auto-detect codec",
					map[string]any{"format": format})
				return
			}

			log.Printf("[preview] asset=%s topic=%s codec=%s format=%q", assetID, topic, codec, format)
			activeCodec = codec
			var initErr error
			var initBuf bytes.Buffer
			if codec == "h265" {
				vps, sps, pps, ok := remux.ExtractVPSPPSHEVC(annexB)
				if !ok {
					httpresp.Error(c, http.StatusUnprocessableEntity,
						codePreviewNoSPSPPS,
						"first IDR sample lacks VPS/SPS/PPS NALUs for h265",
						nil)
					return
				}
				initErr = muxer.WriteInitHEVC(&initBuf, [][]byte{vps}, [][]byte{sps}, [][]byte{pps})
			} else {
				sps, pps, ok := remux.ExtractSPSPPS(annexB)
				if !ok {
					httpresp.Error(c, http.StatusUnprocessableEntity,
						codePreviewNoSPSPPS,
						"first IDR sample lacks SPS/PPS NALUs for h264",
						nil)
					return
				}
				initErr = muxer.WriteInit(&initBuf, [][]byte{sps}, [][]byte{pps})
			}
			if initErr != nil {
				httpresp.Internal(c, fmt.Sprintf("write init segment: %v", initErr))
				return
			}
			c.Writer.Header().Set("Content-Type", "video/mp4")
			c.Writer.Header().Set("Cache-Control", "no-store")
			c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
			// Hint downstream nginx/gateway to stream chunks immediately.
			c.Writer.Header().Set("X-Accel-Buffering", "no")
			c.Writer.Header().Set("X-Preview-Codec", codec)
			c.Writer.WriteHeader(http.StatusOK)
			w, writeErr := c.Writer.Write(initBuf.Bytes())
			bytesWritten += uint64(w)
			if writeErr != nil {
				log.Printf(
					"[preview] asset=%s topic=%s init_write_error bytes=%d err=%v",
					assetID, topic, bytesWritten, writeErr,
				)
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			headerWritten = true
			lastLogTimeNs = m.LogTime
			haveLast = true
			pending = append(pending, remux.Sample{
				AnnexB:        annexB,
				DurationTicks: 0, // patched when next sample arrives
				IsKeyframe:    isKey,
			})
			pendingBytes += uint64(len(annexB))
			totalSamples++
			continue
		}

		// Header written: compute the previous sample's duration from
		// the gap to this message, then either continue accumulating or
		// flush a fragment.
		nextLastLogTime := m.LogTime
		if haveLast && len(pending) > 0 {
			delta, monotonicLast := monotonicDelta(lastLogTimeNs, m.LogTime)
			nextLastLogTime = monotonicLast
			if m.LogTime <= lastLogTimeNs {
				log.Printf(
					"[preview] asset=%s topic=%s non_monotonic_log_time prev=%d cur=%d",
					assetID, topic, lastLogTimeNs, m.LogTime,
				)
			}
			dur := nsToTicks(delta)
			if dur == 0 {
				dur = fallbackFrameDurTick
			}
			pending[len(pending)-1].DurationTicks = dur
			pendingTotalDur += uint64(dur)
		}

		fragmentTarget := targetFragmentDurTicks
		if !firstCutDone && minFirstFragmentDurTicks > fragmentTarget {
			fragmentTarget = minFirstFragmentDurTicks
		}
		// HEVC fragmented output is less broadly stable across browser pipelines.
		// Keep a single growing fragment for h265 and flush on EOF/guard only.
		if activeCodec != "h265" && isKey && pendingTotalDur >= fragmentTarget {
			if ok := finalize("keyframe_threshold"); !ok {
				return
			}
			firstCutDone = true
		}

		pending = append(pending, remux.Sample{
			AnnexB:     annexB,
			IsKeyframe: isKey,
		})
		pendingBytes += uint64(len(annexB))
		if shouldForceFlushPending(len(pending), pendingTotalDur, pendingBytes) {
			pending[len(pending)-1].DurationTicks = fallbackFrameDurTick
			pendingTotalDur += fallbackFrameDurTick
			if ok := finalize("pending_guard"); !ok {
				return
			}
			firstCutDone = true
		}
		totalSamples++
		lastLogTimeNs = nextLastLogTime
		haveLast = true
	}

	// End of stream: pad the last sample with the previously-observed
	// inter-frame interval so the trailing fragment is well-formed.
	if len(pending) > 0 {
		pending[len(pending)-1].DurationTicks = fallbackFrameDurTick
	}
	if ok := finalize("eof"); !ok {
		return
	}
	log.Printf(
		"[preview] asset=%s topic=%s done samples=%d fragments=%d bytes=%d",
		assetID, topic, totalSamples, fragmentCount, bytesWritten,
	)
}

func streamSegmentsViaFFmpeg(
	c *gin.Context,
	topic string,
	startNs, endNs uint64,
	mode string,
	clipSeconds int64,
	openForTranscode func(ctx context.Context) (io.ReadSeeker, func(), error),
) {
	assetID := c.Param("id")
	cacheKey := previewMP4Cache.key(assetID, topic, strconv.FormatUint(startNs, 10), strconv.FormatUint(endNs, 10), "h265-to-h264-stream-v3")

	setCommonHeaders := func(codecSource, acceptRanges, cacheControl string) {
		c.Writer.Header().Set("Content-Type", "video/mp4")
		c.Writer.Header().Set("Accept-Ranges", acceptRanges)
		c.Writer.Header().Set("Cache-Control", cacheControl)
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Accel-Buffering", "no")
		c.Writer.Header().Set("X-Preview-Codec", "h264")
		c.Writer.Header().Set("X-Preview-Codec-Source", codecSource)
		c.Writer.Header().Set("X-Preview-Topic", topic)
		c.Writer.Header().Set("X-Preview-Window-StartNs", strconv.FormatUint(startNs, 10))
		c.Writer.Header().Set("X-Preview-Window-EndNs", strconv.FormatUint(endNs, 10))
		c.Writer.Header().Set("X-Preview-Mode", mode)
		c.Writer.Header().Set("X-Preview-Clip-Seconds", strconv.FormatInt(clipSeconds, 10))
	}

	// Cache hit: serve the pre-built fragmented MP4 with full Range support
	// so subsequent loads behave like a normal static asset.
	if path, ok := previewMP4Cache.get(cacheKey); ok {
		f, err := os.Open(path)
		if err == nil {
			defer f.Close()
			if st, sterr := f.Stat(); sterr == nil {
				setCommonHeaders("h265-transcoded", "bytes", "private, max-age=300")
				http.ServeContent(c.Writer, c.Request, filepath.Base(path), st.ModTime(), f)
				return
			}
		}
		// fall through and re-build on stat/open failure.
	}

	// Cache miss: stream ffmpeg output to the client progressively, and tee
	// the same bytes to a tmp file so subsequent visits become cache hits.
	//
	// The previous design ran the transcode to completion before sending any
	// bytes (build-to-cache-then-serve). That gave perfectly smooth playback
	// at the cost of ~30–90 s of first-byte latency on long clips — a bad
	// trade for users who only want to peek at the first few seconds. The
	// fragmented-MP4 output (frag_keyframe+empty_moov+default_base_moof) is
	// playable as bytes arrive, so we ship it directly. Occasional `waiting`
	// stalls if ffmpeg falls briefly behind are still nicer than a minute of
	// dead air.
	setCommonHeaders("h265-transcoded-streaming", "none", "no-store")
	c.Writer.WriteHeader(http.StatusOK)

	tmpFile, tmpPath, terr := previewMP4Cache.newTempFile(cacheKey)
	var out io.Writer = c.Writer
	if terr == nil {
		out = io.MultiWriter(c.Writer, tmpFile)
	} else {
		log.Printf("[preview] asset=%s topic=%s cache_tmp_err=%v (streaming uncached)", assetID, topic, terr)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()
	transcodeErr := transcodeWindowToWriter(ctx, openForTranscode, topic, startNs, endNs, out)

	if tmpFile != nil {
		_ = tmpFile.Close()
		if transcodeErr == nil {
			// Commit + faststart-remux in the background; the response is
			// already on the wire so this only benefits future visits.
			go func(path string) {
				if _, cerr := previewMP4Cache.commit(cacheKey, path); cerr != nil {
					log.Printf("[preview] asset=%s topic=%s cache_commit_err=%v", assetID, topic, cerr)
					_ = os.Remove(path)
					return
				}
				faststartRemux(assetID, topic, cacheKey)
			}(tmpPath)
		} else {
			_ = os.Remove(tmpPath)
		}
	}

	if transcodeErr != nil && !errors.Is(transcodeErr, context.Canceled) {
		// Headers are already flushed; the client will see a truncated MP4.
		// Browsers handle this gracefully (the <video> ends playback at the
		// last good fragment). Log for observability.
		log.Printf("[preview] asset=%s topic=%s stream_err=%v", assetID, topic, transcodeErr)
	}
}

// faststartRemux runs ffmpeg in copy mode to convert a freshly-committed
// fragmented MP4 into a faststart-friendly non-fragmented MP4. Failures are
// logged and the original (working but slow on cache hit) file is kept.
func faststartRemux(assetID, topic, cacheKey string) {
	src := previewMP4Cache.finalPath(cacheKey)
	tmp := src + ".faststart.tmp"
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", src,
		"-c", "copy",
		"-movflags", "+faststart",
		"-f", "mp4", tmp,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[preview] asset=%s topic=%s faststart_err=%v out=%s", assetID, topic, err, string(out))
		_ = os.Remove(tmp)
		return
	}
	if err := os.Rename(tmp, src); err != nil {
		log.Printf("[preview] asset=%s topic=%s faststart_rename_err=%v", assetID, topic, err)
		_ = os.Remove(tmp)
		return
	}
	log.Printf("[preview] asset=%s topic=%s faststart_ok cache=%s", assetID, topic, filepath.Base(src))
}

// transcodeWindowToWriter pipes HEVC samples from the MCAP window to
// ffmpeg's stdin and copies ffmpeg's fragmented-MP4 stdout into out as
// fast as ffmpeg emits it.
//
// We use a single attempt with a 5s preroll: this matches the previous
// "fast" attempt and is sufficient for HEVC keyframes that carry
// parameter sets in-band. Once we start writing bytes to the response we
// can't retry anyway, so the file-cache path (still available via prewarm)
// is the right place to add more aggressive recovery if we need it.
func transcodeWindowToWriter(
	ctx context.Context,
	openForTranscode func(ctx context.Context) (io.ReadSeeker, func(), error),
	topic string,
	startNs, endNs uint64,
	out io.Writer,
) error {
	const prerollNs uint64 = 5_000_000_000
	rs, closer, err := openForTranscode(ctx)
	if err != nil {
		return fmt.Errorf("open mcap: %w", err)
	}
	if closer != nil {
		defer closer()
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek mcap: %w", err)
	}
	r, err := mcap.NewReader(rs)
	if err != nil {
		return fmt.Errorf("mcap reader: %w", err)
	}
	defer r.Close()

	seekStart := uint64(0)
	if startNs > prerollNs {
		seekStart = startNs - prerollNs
	}
	opts := []mcap.ReadOpt{mcap.WithTopics([]string{topic})}
	if seekStart > 0 {
		opts = append(opts, mcap.AfterNanos(seekStart))
	}
	if endNs > 0 {
		opts = append(opts, mcap.BeforeNanos(endNs))
	}
	it, err := r.Messages(opts...)
	if err != nil {
		return fmt.Errorf("mcap messages: %w", err)
	}

	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-y",
		"-hide_banner", "-loglevel", "error",
		"-fflags", "+genpts+discardcorrupt",
		"-err_detect", "ignore_err",
		"-probesize", "256k",
		"-analyzeduration", "1000000",
		"-f", "hevc", "-i", "pipe:0",
		"-an",
		// Ultra-fast preview profile: aggressively shrink spatial+temporal load
		// to minimize cold-build latency. Source is typically 3840x1200@30fps;
		// 640-wide @8fps reduces pixel throughput by ~33x.
		"-vf", "scale=640:-2,fps=8",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-crf", "35",
		"-g", "8",
		"-keyint_min", "8",
		"-sc_threshold", "0",
		"-x264-params", "bframes=0:rc-lookahead=0",
		"-flush_packets", "1",
		"-muxdelay", "0",
		"-muxpreload", "0",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-f", "mp4", "pipe:1",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout: %w", err)
	}
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}
	// Copy ffmpeg's fragmented-MP4 output to the caller-supplied sink in
	// parallel with the feeder goroutine. We collect any copy error and
	// surface it alongside the feed/wait errors below.
	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(out, stdout)
		copyErrCh <- err
	}()

	go func() {
		if stderr == nil {
			return
		}
		b, _ := io.ReadAll(stderr)
		if len(b) > 0 {
			log.Printf("[preview] topic=%s ffmpeg_stderr=%s", topic, strings.TrimSpace(string(b)))
		}
	}()

	var (
		msg               mcap.Message
		started           bool
		candidateKeyframe []byte
		candidateLogTime  uint64
		lastVPS           []byte
		lastSPS           []byte
		lastPPS           []byte
		writtenFrames     int
	)

	writeErr := make(chan error, 1)
	go func() {
		defer stdin.Close()
		for {
			_, ch, m, err := it.NextInto(&msg)
			if err != nil {
				if errors.Is(err, io.EOF) {
					writeErr <- nil
					return
				}
				writeErr <- err
				return
			}
			if m == nil || ch == nil || ch.Topic != topic {
				continue
			}
			annexB, _, derr := remux.DecodeFoxgloveCompressedVideo(m.Data)
			if derr != nil {
				continue
			}
			annexB = remux.NormalizeToAnnexB(annexB)
			if len(annexB) == 0 {
				continue
			}
			isKey := remux.IsAnnexBKeyframeHEVC(annexB)
			vps, sps, pps, hasPS := remux.ExtractVPSPPSHEVC(annexB)
			if hasPS {
				lastVPS = append(lastVPS[:0], vps...)
				lastSPS = append(lastSPS[:0], sps...)
				lastPPS = append(lastPPS[:0], pps...)
			}
			if !started {
				if m.LogTime < startNs {
					if isKey && hasPS {
						candidateKeyframe = append(candidateKeyframe[:0], annexB...)
						candidateLogTime = m.LogTime
					}
					continue
				}
				if len(candidateKeyframe) > 0 {
					if _, err := stdin.Write(candidateKeyframe); err != nil {
						writeErr <- err
						return
					}
					started = true
					writtenFrames++
					log.Printf("[preview] topic=%s hevc_preroll_keyframe log_time=%d", topic, candidateLogTime)
				}
				if !started {
					if !(isKey && hasPS) {
						continue
					}
					started = true
					if len(lastVPS) > 0 && len(lastSPS) > 0 && len(lastPPS) > 0 {
						if _, err := stdin.Write(lastVPS); err != nil {
							writeErr <- err
							return
						}
						if _, err := stdin.Write(lastSPS); err != nil {
							writeErr <- err
							return
						}
						if _, err := stdin.Write(lastPPS); err != nil {
							writeErr <- err
							return
						}
					}
				}
			}
			if _, err := stdin.Write(annexB); err != nil {
				writeErr <- err
				return
			}
			writtenFrames++
		}
	}()

	feedErr := <-writeErr
	copyErr := <-copyErrCh
	waitErr := cmd.Wait()
	if feedErr != nil && !errors.Is(feedErr, context.Canceled) {
		log.Printf("[preview] topic=%s ffmpeg_feed_err=%v", topic, feedErr)
	}
	if copyErr != nil && !errors.Is(copyErr, context.Canceled) {
		log.Printf("[preview] topic=%s ffmpeg_copy_err=%v", topic, copyErr)
	}
	if waitErr != nil && !errors.Is(waitErr, context.Canceled) {
		log.Printf("[preview] topic=%s ffmpeg_wait_err=%v", topic, waitErr)
	}
	if feedErr != nil {
		return feedErr
	}
	if copyErr != nil {
		return copyErr
	}
	if waitErr != nil {
		return waitErr
	}
	if writtenFrames == 0 {
		return errors.New("no frames written to ffmpeg")
	}
	return nil
}

func normalizeCodec(format string) (string, bool) {
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

// nsToTicks converts a positive nanosecond delta to 90 kHz ticks.
func nsToTicks(deltaNs uint64) uint32 {
	if deltaNs == 0 {
		return 0
	}
	t := deltaNs * uint64(videoTimescale) / 1_000_000_000
	if t == 0 {
		return 1
	}
	if t > 0xFFFFFFFF {
		return 0xFFFFFFFF
	}
	return uint32(t)
}

func monotonicDelta(last, current uint64) (delta uint64, nextLast uint64) {
	if current > last {
		return current - last, current
	}
	return 0, last
}

func shouldForceFlushPending(sampleCount int, pendingDurTicks, pendingBytes uint64) bool {
	return sampleCount >= maxPendingSamples ||
		pendingDurTicks >= maxPendingDurTicks ||
		pendingBytes >= maxPendingBytes
}
