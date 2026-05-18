package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/services/mcap-preview/internal/remux"
	"github.com/foxglove/mcap/go/mcap"
)

func main() {
	var (
		in    = flag.String("in", "", "input mcap path")
		topic = flag.String("topic", "", "topic to extract")
		out   = flag.String("out", "", "output elementary stream path (.h264/.h265)")
		limit = flag.Int("limit", 0, "max frames (0=all)")
	)
	flag.Parse()
	if *in == "" {
		panic("-in required")
	}
	f, err := os.Open(*in)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r, err := mcap.NewReader(f)
	if err != nil {
		panic(err)
	}
	defer r.Close()
	info, err := r.Info()
	if err != nil {
		panic(err)
	}

	if *topic == "" {
		type cand struct{ t, schema string }
		var cands []cand
		for _, ch := range info.Channels {
			if ch == nil {
				continue
			}
			s := info.Schemas[ch.SchemaID]
			if s == nil {
				continue
			}
			if strings.Contains(strings.ToLower(s.Name), "compressedvideo") {
				cands = append(cands, cand{t: ch.Topic, schema: s.Name})
			}
		}
		sort.Slice(cands, func(i, j int) bool { return cands[i].t < cands[j].t })
		fmt.Println("compressed topics:")
		for _, c := range cands {
			fmt.Printf("- %s (%s)\n", c.t, c.schema)
		}
		return
	}

	it, err := r.Messages(mcap.WithTopics([]string{*topic}))
	if err != nil {
		panic(err)
	}
	msg := mcap.Message{}
	var fo *os.File
	if *out != "" {
		fo, err = os.Create(*out)
		if err != nil {
			panic(err)
		}
		defer fo.Close()
	}

	count := 0
	codecCount := map[string]int{}
	for {
		_, ch, m, err := it.NextInto(&msg)
		if err != nil || m == nil {
			break
		}
		if ch == nil || ch.Topic != *topic {
			continue
		}
		data, format, derr := remux.DecodeFoxgloveCompressedVideo(m.Data)
		if derr != nil {
			continue
		}
		codecCount[strings.ToLower(strings.TrimSpace(format))]++
		if fo != nil {
			data = remux.NormalizeToAnnexB(data)
			if _, werr := fo.Write(data); werr != nil {
				panic(werr)
			}
		}
		count++
		if *limit > 0 && count >= *limit {
			break
		}
	}
	fmt.Printf("topic=%s frames=%d codecs=%v out=%s\n", *topic, count, codecCount, *out)
}

