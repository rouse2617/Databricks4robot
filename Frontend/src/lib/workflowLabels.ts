/** Argo system labels that duplicate status columns or add no user value. */
const HIDDEN_LABEL_KEYS = new Set([
	"workflows.argoproj.io/completed",
	"workflows.argoproj.io/creator",
	"workflows.argoproj.io/phase",
]);

const LABEL_KEY_ALIASES: Record<string, string> = {
	"workflows.argoproj.io/resubmitted-from-workflow": "重提交自",
};

export function formatWorkflowLabelKey(key: string): string {
	return (
		LABEL_KEY_ALIASES[key] ?? key.replace(/^workflows\.argoproj\.io\//, "")
	);
}

export function getDisplayLabelEntries(
	labels?: Record<string, string>,
): Array<[string, string]> {
	return Object.entries(labels ?? {}).filter(
		([key]) => !HIDDEN_LABEL_KEYS.has(key),
	);
}

export function serializeWorkflowLabel(key: string, value: string): string {
	return `${key}=${value}`;
}
