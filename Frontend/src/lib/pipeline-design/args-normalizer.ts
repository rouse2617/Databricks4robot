import type { Argument } from "../../components/pipeline/types";

/** Coerce API/canvas args (string[] or Argument[]) into transpiler Argument objects. */
export function normalizeComponentArgs(
	args: unknown[] | undefined,
): Argument[] {
	if (!args || args.length === 0) return [];
	return args.map((item, index) => {
		if (typeof item === "string") {
			const value = item.trim();
			return { name: value || `arg${index + 1}`, value };
		}
		if (item && typeof item === "object") {
			const record = item as { name?: string; value?: string; from?: string };
			const value = record.value ?? record.name ?? "";
			const name = record.name?.trim() || value || `arg${index + 1}`;
			return record.from ? { name, value, from: record.from } : { name, value };
		}
		return { name: `arg${index + 1}`, value: String(item) };
	});
}

function isShellBinary(value: string) {
	return [
		"sh",
		"bash",
		"dash",
		"zsh",
		"/bin/sh",
		"/bin/bash",
		"/usr/bin/sh",
		"/usr/bin/bash",
	].includes(value.trim());
}

export function normalizeShellCommandArgs(
	command: string[],
	args: Argument[],
): Argument[] {
	if (
		command.length !== 2 ||
		!isShellBinary(command[0]) ||
		command[1] !== "-c" ||
		args.length === 0
	) {
		return args;
	}
	const values = args.map((arg) => arg.value?.trim() || "");
	if (values.length >= 3 && isShellBinary(values[0]) && values[1] === "-c") {
		const last = args[args.length - 1];
		return [{ name: last.name || "script", value: last.value || "" }];
	}
	if (values.length >= 2 && values[0] === "-c") {
		const last = args[args.length - 1];
		return [{ name: last.name || "script", value: last.value || "" }];
	}
	return args;
}
