import { describe, expect, it } from "vitest";
import {
	dedupePipelineComponentsByName,
	formatComponentImage,
} from "./pipelineComponentDisplay";

describe("formatComponentImage", () => {
	it("does not append tag when image already has one", () => {
		expect(formatComponentImage("busybox:latest", "latest")).toBe(
			"busybox:latest",
		);
	});

	it("appends tag when image has no tag", () => {
		expect(formatComponentImage("busybox", "latest")).toBe("busybox:latest");
	});
});

describe("dedupePipelineComponentsByName", () => {
	it("keeps stable system component when names duplicate", () => {
		const items = dedupePipelineComponentsByName([
			{
				id: "0a7df9f5-9371-4e50-af3a-89410fa4420e",
				name: "Pass Through",
				type: "container",
				image: "busybox:latest",
				tag: "latest",
				source: "system",
				description: "Pass through component",
				inputPorts: [],
				outputPorts: [],
				createdAt: "2026-05-27T14:21:40Z",
				updatedAt: "2026-05-27T14:21:40Z",
			},
			{
				id: "sys-pass-through",
				name: "Pass Through",
				type: "container",
				image: "busybox:latest",
				tag: "latest",
				source: "system",
				description: "Pass through component",
				inputPorts: [],
				outputPorts: [],
				createdAt: "2026-05-27T14:32:24Z",
				updatedAt: "2026-05-27T14:32:24Z",
			},
		]);

		expect(items).toHaveLength(1);
		expect(items[0]?.id).toBe("sys-pass-through");
	});
});
