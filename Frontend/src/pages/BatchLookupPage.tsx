// CYB-4333: batch-lookup hub. Wraps `BatchAssetLookup` with a Segmented
// preset picker so `/assets?view=lookup` can host any preset — durations,
// costs, and any future one — without adding a new outer tab per preset.
//
// URL contract:
//   /assets?view=lookup                       → DEFAULT_PRESET_KEY (durations)
//   /assets?view=lookup&preset=durations      → durations preset
//   /assets?view=lookup&preset=costs          → costs preset
//   /assets?view=lookup&preset=<unknown>      → falls back to default
//
// `view` is owned by the workbench (AssetsWorkbenchPage); this component
// only touches `preset` and leaves the rest of the query string alone.

import { Segmented } from "antd";
import { useSearchParams } from "react-router-dom";
import BatchAssetLookup from "../components/batch-lookup/BatchAssetLookup";
import {
	BATCH_LOOKUP_PRESETS,
	PRESET_OPTIONS,
	resolvePresetKey,
} from "./presets";

export default function BatchLookupPage() {
	const [params, setParams] = useSearchParams();
	const activeKey = resolvePresetKey(params.get("preset"));

	function handleChange(next: string | number): void {
		setParams(
			(prev) => {
				const out = new URLSearchParams(prev);
				out.set("preset", String(next));
				return out;
			},
			// replace so preset clicks don't flood browser history.
			{ replace: true },
		);
	}

	return (
		<div>
			<div style={{ padding: "12px 24px 0" }}>
				<Segmented
					value={activeKey}
					onChange={handleChange}
					options={PRESET_OPTIONS.map((o) => ({
						value: o.key,
						label: (
							<span>
								{o.icon} {o.label}
							</span>
						),
					}))}
					data-testid="batch-lookup-preset-picker"
				/>
			</div>
			<BatchAssetLookup preset={BATCH_LOOKUP_PRESETS[activeKey]} />
		</div>
	);
}
