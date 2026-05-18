/**
 * Unified formatting utilities for the Cyber Databrew frontend.
 * All components should use these instead of inline dayjs/Date formatting.
 */

import dayjs from "dayjs";

/** Timestamp → "YYYY-MM-DD HH:mm" — primary format for all data tables */
export function formatDateTime(
	value: string | number | Date | null | undefined,
): string {
	if (value == null) return "—";
	const d = dayjs(value);
	return d.isValid() ? d.format("YYYY-MM-DD HH:mm") : "—";
}

/** Timestamp → "YYYY-MM-DD" — compact date-only format */
export function formatDate(
	value: string | number | Date | null | undefined,
): string {
	if (value == null) return "—";
	const d = dayjs(value);
	return d.isValid() ? d.format("YYYY-MM-DD") : "—";
}

/** Timestamp → "HH:mm:ss" — compact time-only format */
export function formatTime(
	value: string | number | Date | null | undefined,
): string {
	if (value == null) return "—";
	const d = dayjs(value);
	return d.isValid() ? d.format("HH:mm:ss") : "—";
}

/** Timestamp → relative human-readable (e.g., "2 分钟前") */
export function formatRelativeTime(
	value: string | number | Date | null | undefined,
): string {
	if (value == null) return "—";
	const d = dayjs(value);
	if (!d.isValid()) return "—";
	const now = dayjs();
	const diffSec = now.diff(d, "second");
	if (diffSec < 60) return `${diffSec} 秒前`;
	const diffMin = now.diff(d, "minute");
	if (diffMin < 60) return `${diffMin} 分钟前`;
	const diffHour = now.diff(d, "hour");
	if (diffHour < 24) return `${diffHour} 小时前`;
	const diffDay = now.diff(d, "day");
	if (diffDay < 7) return `${diffDay} 天前`;
	return d.format("MM-DD");
}

/** Number → locale-formatted string (e.g., "268,871") */
export function formatNumber(value: number | null | undefined): string {
	if (value == null) return "—";
	return value.toLocaleString();
}
