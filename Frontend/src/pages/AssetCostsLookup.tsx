// CYB-4306: 5-line shim that mounts the shared BatchAssetLookup with the
// costs preset. Mirrors the AssetDurationLookup shim shipped in CYB-4304
// for lazy-chunk parity — the workbench tab lazy-imports this file.
import BatchAssetLookup from "../components/batch-lookup/BatchAssetLookup";
import { costsPreset } from "./presets/costs";

export default function AssetCostsLookup() {
	return <BatchAssetLookup preset={costsPreset} />;
}
