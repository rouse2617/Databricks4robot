import { describe, expect, it } from "vitest";
import {
	buildSegmentPreviewUrl,
	enrichSegmentPreviewUrl,
	isHevcPreviewCodec,
} from "./previewSegmentUrl";

describe("enrichSegmentPreviewUrl", () => {
	it("adds window and grace_token query params", () => {
		Object.defineProperty(document, "cookie", {
			configurable: true,
			get: () => "grace_session=dev-token",
		});
		const url = enrichSegmentPreviewUrl(
			"/api/v1/preview/assets/abc/segment.mp4?topic=%2Fcam",
			{ startNs: 1000, endNs: 2000 },
		);
		expect(url).toContain("grace_token=dev-token");
		expect(url).toContain("start_ns=1000");
		expect(url).toContain("end_ns=2000");
		expect(url).toContain("topic=%2Fcam");
	});
});

describe("buildSegmentPreviewUrl", () => {
	it("includes topic when provided", () => {
		const url = buildSegmentPreviewUrl("abc", { topic: "/camera/front" });
		expect(url).toContain("topic=");
	});
});

describe("isHevcPreviewCodec", () => {
	it("detects h265 and hevc", () => {
		expect(isHevcPreviewCodec("h265")).toBe(true);
		expect(isHevcPreviewCodec("hevc")).toBe(true);
		expect(isHevcPreviewCodec("h264")).toBe(false);
	});
});
