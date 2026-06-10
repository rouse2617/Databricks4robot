/** Matches backend + OpenAPI RunID: 16 alphanumeric ASCII. */
const RUN_ID_RE = /^[0-9A-Za-z]{16}$/;

export function isRegisteredRunId(id: string | undefined | null): id is string {
	return typeof id === "string" && RUN_ID_RE.test(id.trim());
}

/** Display form: first 4 + ellipsis + last 4. */
export function formatRunIdShort(id: string): string {
	if (id.length <= 12) return id;
	return `${id.slice(0, 4)}…${id.slice(-4)}`;
}
