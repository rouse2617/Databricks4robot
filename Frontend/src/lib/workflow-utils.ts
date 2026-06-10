const URL_PATTERN = /((?:https?:\/\/|www\.)[^\s<>"']+)/gi;

export function formatDuration(
	startedAt?: string,
	finishedAt?: string,
): string {
	if (!startedAt) return "-";

	const start = new Date(startedAt).getTime();
	const end = finishedAt ? new Date(finishedAt).getTime() : Date.now();
	if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) {
		return "-";
	}

	const totalSeconds = Math.floor((end - start) / 1000);
	const hours = Math.floor(totalSeconds / 3600);
	const minutes = Math.floor((totalSeconds % 3600) / 60);
	const seconds = totalSeconds % 60;

	if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`;
	if (minutes > 0) return `${minutes}m ${seconds}s`;
	return `${seconds}s`;
}

export function splitLinkifiedText(text: string) {
	const parts: Array<{
		type: "text" | "link";
		text: string;
		start: number;
		href?: string;
	}> = [];
	let lastIndex = 0;

	for (const match of text.matchAll(URL_PATTERN)) {
		const index = match.index ?? 0;
		if (index > lastIndex) {
			parts.push({
				type: "text",
				text: text.slice(lastIndex, index),
				start: lastIndex,
			});
		}

		const value = match[0];
		parts.push({
			type: "link",
			text: value,
			start: index,
			href: value.startsWith("http") ? value : `https://${value}`,
		});
		lastIndex = index + value.length;
	}

	if (lastIndex < text.length) {
		parts.push({ type: "text", text: text.slice(lastIndex), start: lastIndex });
	}

	return parts.length > 0 ? parts : [{ type: "text" as const, text, start: 0 }];
}
