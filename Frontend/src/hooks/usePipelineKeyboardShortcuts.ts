import { useEffect } from "react";

export interface PipelineKeyboardShortcutHandlers {
	onSave: () => void;
	onDeploy: () => void;
	onClear: () => void;
	onCloseModal?: () => void;
	canDeploy: boolean;
	modalOpen: boolean;
}

function isEditableTarget(target: EventTarget | null): boolean {
	if (!(target instanceof HTMLElement)) return false;
	const tag = target.tagName;
	return (
		target.isContentEditable ||
		tag === "INPUT" ||
		tag === "TEXTAREA" ||
		tag === "SELECT"
	);
}

export function usePipelineKeyboardShortcuts({
	onSave,
	onDeploy,
	onClear,
	onCloseModal,
	canDeploy,
	modalOpen,
}: PipelineKeyboardShortcutHandlers): void {
	useEffect(() => {
		const onKeyDown = (event: KeyboardEvent) => {
			if (isEditableTarget(event.target)) return;

			const mod = event.metaKey || event.ctrlKey;

			if (event.key === "Escape" && modalOpen) {
				event.preventDefault();
				onCloseModal?.();
				return;
			}

			if (!mod) return;

			const key = event.key.toLowerCase();
			if (key === "s") {
				event.preventDefault();
				onSave();
				return;
			}
			if (key === "d" && canDeploy) {
				event.preventDefault();
				onDeploy();
				return;
			}
			if ((event.key === "Backspace" || event.key === "Delete") && !modalOpen) {
				event.preventDefault();
				onClear();
			}
		};

		window.addEventListener("keydown", onKeyDown);
		return () => window.removeEventListener("keydown", onKeyDown);
	}, [canDeploy, modalOpen, onClear, onCloseModal, onDeploy, onSave]);
}
