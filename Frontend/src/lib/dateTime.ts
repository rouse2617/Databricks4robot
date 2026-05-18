import dayjs from "dayjs";

export function formatDateTime(value?: string | null): string {
	if (!value) return "—";
	const parsed = dayjs(value);
	return parsed.isValid() ? parsed.format("YYYY-MM-DD HH:mm:ss") : "—";
}

export function formatShortDateTime(value?: string | null): string {
	if (!value) return "—";
	const parsed = dayjs(value);
	return parsed.isValid() ? parsed.format("MM-DD HH:mm") : "—";
}
