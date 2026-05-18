import { useEffect } from "react";
import type { Asset } from "../../api/types";

interface UseAssetsHotkeysArgs {
	containerRef: React.RefObject<HTMLDivElement | null>;
	items: Asset[];
	activeAssetId: string | null;
	onSetActiveAsset: (assetId: string) => void;
	onToggleActiveSelection: () => void;
	onCloseOverlays: () => void;
}

export function useAssetsHotkeys({
	containerRef,
	items,
	activeAssetId,
	onSetActiveAsset,
	onToggleActiveSelection,
	onCloseOverlays,
}: UseAssetsHotkeysArgs): void {
	useEffect(() => {
		const root = containerRef.current;
		if (!root) return;

		const handleKeyDown = (e: KeyboardEvent) => {
			const target = e.target as HTMLElement | null;
			if (!target || !root.contains(target)) return;

			const tag = target.tagName;
			const isInInput =
				tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
			const isInInteractive =
				isInInput ||
				tag === "BUTTON" ||
				tag === "A" ||
				tag === "VIDEO" ||
				tag === "AUDIO" ||
				target.isContentEditable ||
				Boolean(
					target.closest(
						'video,audio,[role="slider"],[role="button"],[role="textbox"],.ant-select,.ant-picker,.ant-input,.ant-slider',
					),
				);

			if (e.key === "/" && !isInInteractive) {
				e.preventDefault();
				const searchInput = root.querySelector<HTMLInputElement>(
					'[data-testid="assets-search-input"]',
				);
				searchInput?.focus();
				return;
			}

			if (e.key === "Escape") {
				onCloseOverlays();
				return;
			}

			if (isInInteractive) return;

			if (e.key === " ") {
				e.preventDefault();
				onToggleActiveSelection();
				return;
			}

			if (e.key !== "ArrowUp" && e.key !== "ArrowDown") return;
			if (items.length === 0) return;

			const currentIdx = activeAssetId
				? items.findIndex((a) => a.asset_id === activeAssetId)
				: -1;
			const nextIdx =
				e.key === "ArrowDown"
					? currentIdx < items.length - 1
						? currentIdx + 1
						: currentIdx
					: currentIdx > 0
						? currentIdx - 1
						: 0;

			const nextAsset = items[nextIdx];
			if (nextAsset && nextAsset.asset_id !== activeAssetId) {
				e.preventDefault();
				onSetActiveAsset(nextAsset.asset_id);
			}
		};

		root.addEventListener("keydown", handleKeyDown);
		return () => root.removeEventListener("keydown", handleKeyDown);
	}, [
		containerRef,
		items,
		activeAssetId,
		onSetActiveAsset,
		onToggleActiveSelection,
		onCloseOverlays,
	]);
}
