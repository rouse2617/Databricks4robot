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

	// CYB-3573: the argo node id (`<workflow>.<node>`, contains a dot) can leak
	// into the podName slot; deep-linking it 404s in the console, so reject it.
	it("returns null for a dotted argo node id (not a real pod name)", () => {
		expect(
			gkePodConsoleUrl(
				"cyber-delivery-prod",
				"c3f710ec-e5b6-4e41-96ad-e89f8e50fbda.step-cc2-mcap-slimmer",
			),
		).toBeNull();
	});

	it("builds a URL for a valid v2 pod name (hyphens + numeric suffix)", () => {
		expect(
			gkePodConsoleUrl(
				"cyber-delivery-prod",
				"c3f710ec-e5b6-4e41-96ad-e89f8e50fbda-step-cc2-mcap-slimmer-516831561",
			),
		).toContain(
			"/cyber-delivery-prod/c3f710ec-e5b6-4e41-96ad-e89f8e50fbda-step-cc2-mcap-slimmer-516831561/details?project=cyberorigin-delivery",
		);
	});

	it("rejects pod names with spaces or other invalid characters", () => {
		expect(
			gkePodConsoleUrl("cyber-delivery-prod", "pod with space"),
		).toBeNull();
		expect(gkePodConsoleUrl("cyber-delivery-prod", "UPPER")).toBeNull();
	});
});
