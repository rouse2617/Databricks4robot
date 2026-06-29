export interface PreviewChannel {
  topic: string;
  label: string;
  width: number;
  height: number;
}

export interface PreviewManifest {
  asset_id: string;
  source_type: "mcap" | "mp4";
  /** Total duration in nanoseconds */
  duration_ns: number;
  channels: PreviewChannel[];
}

export const MOCK_MANIFESTS: Record<string, PreviewManifest> = {
  "test-001": {
    asset_id: "test-001",
    source_type: "mcap",
    duration_ns: 948_340_000_000,
    channels: [
      {
        topic: "/camera/front/image_raw/compressed",
        label: "front",
        width: 3840,
        height: 1200,
      },
      {
        topic: "/camera/side/image_raw/compressed",
        label: "side",
        width: 2560,
        height: 960,
      },
      {
        topic: "/camera/down/image_raw/compressed",
        label: "down",
        width: 2560,
        height: 960,
      },
    ],
  },
  "test-002": {
    asset_id: "test-002",
    source_type: "mp4",
    duration_ns: 623_700_000_000,
    channels: [
      {
        topic: "/camera/front/image_raw/compressed",
        label: "front",
        width: 3840,
        height: 1200,
      },
    ],
  },
};

export function getMockManifest(
  assetId: string,
): PreviewManifest | undefined {
  return MOCK_MANIFESTS[assetId];
}
