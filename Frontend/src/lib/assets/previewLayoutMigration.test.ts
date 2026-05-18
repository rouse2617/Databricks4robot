import { describe, expect, it } from "vitest";
import {
	LATEST_PREVIEW_LAYOUT_VERSION,
	migrateLayout,
} from "./previewLayoutMigration";

describe("migrateLayout", () => {
	it("migrates v1 layout to v2 with field rename", () => {
		const migrated = migrateLayout({
			version: 1,
			preview: "asset-1",
			source: "live_topic_0",
			topic: "/camera/front",
			time: 12.5,
			ds: "remote-file",
			dsParams: { url: "https://example.com/a.mcap" },
			previewPanel: {
				sidebarTab: "alerts",
				playbackRate: "1.5",
			},
		});

		expect(migrated.version).toBe(LATEST_PREVIEW_LAYOUT_VERSION);
		expect(migrated.previewPanel.sourceId).toBe("live_topic_0");
		expect(migrated.previewPanel.primaryTopic).toBe("/camera/front");
		expect(migrated.previewPanel.timeSec).toBe(12.5);
		expect(migrated.previewPanel.sidebarTab).toBe("alerts");
		expect(migrated.previewPanel.playbackRate).toBe("1.5");
	});

	it("fills v2 defaults when fields are missing", () => {
		const migrated = migrateLayout({
			version: 1,
			preview: "asset-2",
		});

		expect(migrated.version).toBe(LATEST_PREVIEW_LAYOUT_VERSION);
		expect(migrated.preview).toBe("asset-2");
		expect(migrated.previewPanel.primaryTopic).toBeNull();
		expect(migrated.previewPanel.sourceId).toBeNull();
		expect(migrated.previewPanel.timeSec).toBeNull();
		expect(migrated.previewPanel.sidebarTab).toBe("tracks");
		expect(migrated.previewPanel.playbackRate).toBe("1.0");
		expect(migrated.dsParams).toEqual({});
	});

	it("falls back safely for unknown layout version", () => {
		const migrated = migrateLayout({
			version: 99,
			previewPanel: {
				sourceId: "source-x",
				primaryTopic: "/cam/rear",
				timeSec: 8,
			},
			dsParams: { asset_id: "abc", ignored: "" },
		});

		expect(migrated.version).toBe(LATEST_PREVIEW_LAYOUT_VERSION);
		expect(migrated.previewPanel.sourceId).toBe("source-x");
		expect(migrated.previewPanel.primaryTopic).toBe("/cam/rear");
		expect(migrated.previewPanel.timeSec).toBe(8);
		expect(migrated.dsParams).toEqual({ asset_id: "abc" });
	});
});
