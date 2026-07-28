// CYB-4304: this file now defers all UX to BatchAssetLookup + durations preset.
// Route contract (/assets?view=durations, mounted by AssetsWorkbenchPage's
// lazy import) stays unchanged.
import BatchAssetLookup from "../components/batch-lookup/BatchAssetLookup";
import { durationsPreset } from "./presets/durations";

export default function AssetDurationLookup() {
	return <BatchAssetLookup preset={durationsPreset} />;
}
