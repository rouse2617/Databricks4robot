package manifest

import "testing"

func TestBuildPreviewSources_StableTopicOrder(t *testing.T) {
	topics := []CandidateVideoTopic{
		{Topic: "/camera/side/image_raw/compressed", SchemaName: "foxglove.CompressedVideo"},
		{Topic: "/camera/front/image_raw/compressed", SchemaName: "foxglove.CompressedVideo"},
		{Topic: "/camera/down/image_raw/compressed", SchemaName: "foxglove.CompressedVideo"},
	}

	for run := 0; run < 5; run++ {
		sources, recommended := buildPreviewSources("asset-1", topics, nil)
		if len(sources) != 3 {
			t.Fatalf("run %d: got %d sources", run, len(sources))
		}
		if sources[0].ID != "live_topic_0" || sources[0].Topic != "/camera/front/image_raw/compressed" {
			t.Fatalf("run %d: front source=%+v", run, sources[0])
		}
		if sources[1].ID != "live_topic_1" || sources[1].Topic != "/camera/side/image_raw/compressed" {
			t.Fatalf("run %d: side source=%+v", run, sources[1])
		}
		if sources[2].ID != "live_topic_2" || sources[2].Topic != "/camera/down/image_raw/compressed" {
			t.Fatalf("run %d: down source=%+v", run, sources[2])
		}
		if recommended != "live_topic_0" {
			t.Fatalf("run %d: recommended=%q", run, recommended)
		}
	}
}
