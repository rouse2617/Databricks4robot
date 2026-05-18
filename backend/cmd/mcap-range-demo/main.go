// mcap-range-demo — local or GCS-backed demo: MCAP summary/index + time-window → byte ranges.
//
// - Local: uses os.File (ReadSeeker).
// - gs://: uses cloud.google.com/go/storage Range reads only (no full-object download).
//
// Usage:
//
//	go run ./cmd/mcap-range-demo -mcap /path/to/file.mcap
//	go run ./cmd/mcap-range-demo -mcap gs://bucket/path/to/file.mcap -from-start-s 70 -to-start-s 90
//
// Validates: demo tooling only (not wired to server).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"cloud.google.com/go/storage"
	"github.com/foxglove/mcap/go/mcap"
)

func main() {
	mcapRef := flag.String("mcap", "", "local path or gs://bucket/object to .mcap (indexed / with summary)")
	t1 := flag.Int64("t1", 0, "window start: log time (nanoseconds)")
	t2 := flag.Int64("t2", 0, "window end: log time (nanoseconds), overlap uses chunk intersection")
	fromStartS := flag.Float64("from-start-s", -1, "if >= 0, window is [msg_start + from-start-s, msg_start + to-start-s] in seconds")
	toStartS := flag.Float64("to-start-s", -1, "paired with -from-start-s")
	maxPrint := flag.Int("print-chunks", 8, "max overlapping chunks to print (0 = none)")
	flag.Parse()

	if *mcapRef == "" {
		log.Fatal("flag -mcap is required")
	}

	ctx := context.Background()

	var (
		label   string
		size    int64
		rs      io.ReadSeeker
		cleanup func()
		counter *countingReadSeeker
		gcsRS   *gcsReadSeeker
	)

	if bucket, object, ok := parseGSURI(*mcapRef); ok {
		cl, err := storage.NewClient(ctx)
		if err != nil {
			log.Fatalf("storage client: %v", err)
		}
		cleanup = func() { _ = cl.Close() }
		obj := cl.Bucket(bucket).Object(object)
		attrs, err := obj.Attrs(ctx)
		if err != nil {
			cl.Close()
			log.Fatalf("gcs attrs %s: %v", *mcapRef, err)
		}
		label = *mcapRef
		size = attrs.Size
		gcsRS = newGCSReadSeeker(ctx, obj, attrs.Size)
		rs = newCountingReadSeeker(gcsRS)
		counter = rs.(*countingReadSeeker)
	} else {
		f, err := os.Open(*mcapRef)
		if err != nil {
			log.Fatal(err)
		}
		cleanup = func() { _ = f.Close() }
		st, err := f.Stat()
		if err != nil {
			f.Close()
			log.Fatal(err)
		}
		label = *mcapRef
		size = st.Size()
		counter = newCountingReadSeeker(f)
		rs = counter
	}
	defer cleanup()

	r, err := mcap.NewReader(rs)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	info, err := r.Info()
	if err != nil {
		log.Fatal(err)
	}

	if counter != nil {
		fmt.Println(counter.Summary())
		fmt.Println("(Many small reads + seeks => not a single sequential scan; each GCS read_call ~= one Range GET.)")
		if gcsRS != nil {
			fmt.Println(gcsRS.StatsSummary())
		}
		fmt.Println()
	}

	fmt.Printf("source: %s (%d bytes)\n", label, size)
	if info.Statistics != nil {
		ms := info.Statistics.MessageStartTime
		me := info.Statistics.MessageEndTime
		fmt.Printf("log span (ns): %d .. %d (%.3f s)\n", ms, me, float64(me-ms)/1e9)
		fmt.Printf("message_count: %d\n", info.Statistics.MessageCount)
	}
	fmt.Printf("chunk_indexes: %d\n", len(info.ChunkIndexes))

	if len(info.ChunkIndexes) == 0 {
		log.Fatal("no chunk indexes — cannot map time → byte ranges (file may be unindexed)")
	}

	var wStart, wEnd uint64
	switch {
	case *fromStartS >= 0 && *toStartS >= 0:
		if info.Statistics == nil {
			log.Fatal("-from-start-s requires Statistics in summary")
		}
		base := info.Statistics.MessageStartTime
		wStart = base + uint64(*fromStartS*1e9)
		wEnd = base + uint64(*toStartS*1e9)
	case *t1 != 0 || *t2 != 0:
		if *t2 <= *t1 {
			log.Fatal("-t1 must be < -t2")
		}
		wStart = uint64(*t1)
		wEnd = uint64(*t2)
	default:
		fmt.Println("\n(no -t1/-t2 or -from-start-s/-to-start-s; skipping overlap)")
		return
	}

	printOverlap(info, wStart, wEnd, *maxPrint)
}

func printOverlap(info *mcap.Info, wStart, wEnd uint64, maxPrint int) {
	fmt.Printf("\nwindow log time (ns): [%d, %d] (%.6f s)\n", wStart, wEnd, float64(wEnd-wStart)/1e9)

	var (
		nHits    int
		sumChunk uint64
		minOff   uint64 = ^uint64(0)
		maxEnd   uint64
		printed  int
	)

	for i := range info.ChunkIndexes {
		ci := info.ChunkIndexes[i]
		if ci == nil {
			continue
		}
		if ci.MessageEndTime < wStart || ci.MessageStartTime > wEnd {
			continue
		}
		nHits++
		sumChunk += ci.ChunkLength
		if ci.ChunkStartOffset < minOff {
			minOff = ci.ChunkStartOffset
		}
		end := ci.ChunkStartOffset + ci.ChunkLength
		if end > maxEnd {
			maxEnd = end
		}
		if maxPrint > 0 && printed < maxPrint {
			fmt.Printf(
				"  chunk[%d] file_off=%d len=%d msg=[%d,%d]\n",
				i, ci.ChunkStartOffset, ci.ChunkLength, ci.MessageStartTime, ci.MessageEndTime,
			)
			printed++
		}
	}

	fmt.Printf("\noverlap: %d chunks\n", nHits)
	fmt.Printf("sum(chunk_length) = %d bytes (%.2f MiB)\n", sumChunk, float64(sumChunk)/(1024*1024))
	if nHits > 0 {
		merged := maxEnd - minOff
		fmt.Printf("merged contiguous span: [%d, %d) = %d bytes (%.2f MiB)\n",
			minOff, maxEnd, merged, float64(merged)/(1024*1024))
		fmt.Println("\nGCS Range hints (one merged request — only valid if no gaps between chunks):")
		fmt.Printf("  Range: bytes=%d-%d\n", minOff, maxEnd-1)
		fmt.Println("If spans have gaps, issue multiple Range requests per chunk (file_off, len).")
	}
}
