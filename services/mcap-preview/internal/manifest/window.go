package manifest

import "github.com/foxglove/mcap/go/mcap"

// NormalizeWindow converts locator window timestamps to nanoseconds by
// selecting the unit (ns/us/ms/s) that best overlaps MCAP statistics.
// When the locator window is empty or clearly mismatched, it falls back
// to the file's full statistics window.
func NormalizeWindow(rawStart, rawEnd int64, info *mcap.Info) (uint64, uint64, string) {
	if rawStart > 0 && rawEnd > 0 {
		orderedStart := uint64(rawStart)
		orderedEnd := uint64(rawEnd)
		if orderedEnd < orderedStart {
			orderedStart, orderedEnd = orderedEnd, orderedStart
		}
		if info == nil || info.Statistics == nil {
			return orderedStart, orderedEnd, "locator_ns_no_stats"
		}

		statsStart := info.Statistics.MessageStartTime
		statsEnd := info.Statistics.MessageEndTime

		type candidate struct {
			start uint64
			end   uint64
			mode  string
			score uint64
		}
		cands := make([]candidate, 0, 4)
		cands = append(cands, candidate{start: orderedStart, end: orderedEnd, mode: "locator_ns"})
		if start, ok1 := safeMul(orderedStart, 1_000); ok1 {
			if end, ok2 := safeMul(orderedEnd, 1_000); ok2 {
				cands = append(cands, candidate{start: start, end: end, mode: "locator_us"})
			}
		}
		if start, ok1 := safeMul(orderedStart, 1_000_000); ok1 {
			if end, ok2 := safeMul(orderedEnd, 1_000_000); ok2 {
				cands = append(cands, candidate{start: start, end: end, mode: "locator_ms"})
			}
		}
		if start, ok1 := safeMul(orderedStart, 1_000_000_000); ok1 {
			if end, ok2 := safeMul(orderedEnd, 1_000_000_000); ok2 {
				cands = append(cands, candidate{start: start, end: end, mode: "locator_s"})
			}
		}
		if len(cands) == 0 {
			return statsStart, statsEnd, "stats_fallback_overflow"
		}
		best := cands[0]
		for _, c := range cands {
			c.score = overlap(c.start, c.end, statsStart, statsEnd)
			if c.score > best.score {
				best = c
			}
		}
		if best.score > 0 {
			return best.start, best.end, best.mode
		}
		return statsStart, statsEnd, "stats_fallback_mismatch"
	}

	if info != nil && info.Statistics != nil {
		return info.Statistics.MessageStartTime, info.Statistics.MessageEndTime, "stats_fallback_empty"
	}
	return 0, 0, "empty_no_stats"
}

func overlap(aStart, aEnd, bStart, bEnd uint64) uint64 {
	if aEnd < bStart || aStart > bEnd {
		return 0
	}
	start := aStart
	if bStart > start {
		start = bStart
	}
	end := aEnd
	if bEnd < end {
		end = bEnd
	}
	return end - start + 1
}

func safeMul(v, factor uint64) (uint64, bool) {
	if factor == 0 {
		return 0, true
	}
	out := v * factor
	if out/factor != v {
		return 0, false
	}
	return out, true
}
