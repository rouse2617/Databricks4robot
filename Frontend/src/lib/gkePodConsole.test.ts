import { describe, expect, it } from "vitest";
import {
	gkeConsoleCoordsForNamespace,
	gkePodConsoleUrl,
} from "./gkePodConsole";

describe("gkePodConsoleUrl", () => {
	it("builds a delivery-clust console URL for a cyber-delivery namespace", () => {
		const url = gkePodConsoleUrl(
			"cyber-delivery-prod",
			"356e519a-59f2-4fe4-848f-c52c136f38bc-step-cc2-mcap-slimmer-2606944259",
		);
		expect(url).toBe(
			"https://console.cloud.google.com/kubernetes/pod/us-central1/delivery-clust/" +
				"cyber-delivery-prod/" +
				"356e519a-59f2-4fe4-848f-c52c136f38bc-step-cc2-mcap-slimmer-2606944259/details" +
				"?project=cyberorigin-delivery",
		);
	});

	it("maps cyber-clust namespaces to green-valley", () => {
		expect(gkePodConsoleUrl("cyber-databrew-dev", "p1")).toContain(
			"/us-central1/cyber-clust/cyber-databrew-dev/p1/details?project=green-valley-442103",
		);
		expect(gkePodConsoleUrl("video-proc-prod", "p2")).toContain(
			"/cyber-clust/video-proc-prod/",
		);
	});

	it("returns null for an unknown namespace (no wrong-project link)", () => {
		expect(gkePodConsoleUrl("some-other-ns", "p1")).toBeNull();
		expect(gkeConsoleCoordsForNamespace("some-other-ns")).toBeNull();
	});

	it("returns null when namespace or pod is missing", () => {
		expect(gkePodConsoleUrl("", "p1")).toBeNull();
		expect(gkePodConsoleUrl("cyber-delivery-prod", "")).toBeNull();
		expect(gkePodConsoleUrl(undefined, undefined)).toBeNull();
	});

	it("url-encodes namespace and pod segments", () => {
		const url = gkePodConsoleUrl("cyber-delivery-prod", "pod with space");
		expect(url).toContain("pod%20with%20space");
	});
});
