// CYB-4333: batch-lookup preset registry.
//
// Every preset (durations, costs, and any future lookup) is registered here.
// `BatchLookupPage` reads `BATCH_LOOKUP_PRESETS[activeKey]` and hands it to
// the shared `BatchAssetLookup` shell. Adding a preset later is one import
// plus one entry in the map and the options array — the key union grows too.

import { ClockCircleOutlined, DollarOutlined } from "@ant-design/icons";
import type { ReactNode } from "react";
import type { BatchLookupPreset } from "../../components/batch-lookup/types";
import { costsPreset } from "./costs";
import { durationsPreset } from "./durations";

export type BatchLookupPresetKey = "durations" | "costs";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyPreset = BatchLookupPreset<any, any>;

export const BATCH_LOOKUP_PRESETS: Record<BatchLookupPresetKey, AnyPreset> = {
	durations: durationsPreset,
	costs: costsPreset,
};

// Default key when the URL has no `preset` param or an unknown one. Named
// explicitly rather than inferring from `Object.keys(...)` so JS-engine key
// order can't change what the user lands on.
export const DEFAULT_PRESET_KEY: BatchLookupPresetKey = "durations";

export const PRESET_OPTIONS: Array<{
	key: BatchLookupPresetKey;
	label: string;
	icon: ReactNode;
}> = [
	{ key: "durations", label: "时长", icon: <ClockCircleOutlined /> },
	{ key: "costs", label: "成本", icon: <DollarOutlined /> },
];

// Silent fallback — bookmark to an unknown preset lands on the default and
// the hub replaces the URL so a refresh gives a clean state. See
// openspec/changes/CYB-4333-lookup-hub/decisions.md.
export function resolvePresetKey(
	raw: string | null | undefined,
): BatchLookupPresetKey {
	if (raw && raw in BATCH_LOOKUP_PRESETS) {
		return raw as BatchLookupPresetKey;
	}
	return DEFAULT_PRESET_KEY;
}
