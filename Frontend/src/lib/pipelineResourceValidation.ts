export type ResourceField = "cpu" | "memory" | "disk" | "gpu";

const MEMORY_OR_DISK_PATTERN =
	/^[0-9]+(?:\.[0-9]+)?(?:Ki|Mi|Gi|Ti|Pi|Ei|K|M|G|T|P|E)$/;
const CPU_PATTERN = /^(?:[0-9]+(?:\.[0-9]+)?|[0-9]+m)$/;
const GPU_PATTERN = /^[0-9]+$/;

const FIELD_LABEL: Record<ResourceField, string> = {
	cpu: "CPU",
	memory: "内存",
	disk: "磁盘",
	gpu: "GPU",
};

function isPositiveNumber(value: string): boolean {
	const number = Number(value);
	return Number.isFinite(number) && number > 0;
}

function hasPositiveNumericPrefix(value: string): boolean {
	const number = Number.parseFloat(value);
	return Number.isFinite(number) && number > 0;
}

export function validateResourceQuantity(
	field: ResourceField,
	value: unknown,
): string | null {
	if (value === undefined || value === null || String(value).trim() === "") {
		return null;
	}
	const text = String(value).trim();
	switch (field) {
		case "cpu":
			if (
				!CPU_PATTERN.test(text) ||
				!isPositiveNumber(text.endsWith("m") ? text.slice(0, -1) : text)
			) {
				return "CPU 需填写正数或 millicore，例如 500m、1";
			}
			return null;
		case "memory":
			if (
				!MEMORY_OR_DISK_PATTERN.test(text) ||
				!hasPositiveNumericPrefix(text)
			) {
				return "内存必须是大于 0 且带单位的数值，例如 512Mi、1Gi";
			}
			return null;
		case "disk":
			if (
				!MEMORY_OR_DISK_PATTERN.test(text) ||
				!hasPositiveNumericPrefix(text)
			) {
				return "磁盘必须是大于 0 且带单位的数值，例如 1Gi、20Gi";
			}
			return null;
		case "gpu":
			if (!GPU_PATTERN.test(text)) {
				return "GPU 必须是非负整数，例如 0、1、2";
			}
			return null;
		default:
			return null;
	}
}

export function resourceQuantityRule(field: ResourceField) {
	return {
		validator(_: unknown, value: unknown) {
			const error = validateResourceQuantity(field, value);
			return error ? Promise.reject(new Error(error)) : Promise.resolve();
		},
	};
}

export function validateResourceMap(
	resources: Partial<Record<ResourceField, unknown>> | undefined,
	context: string,
): string[] {
	if (!resources) return [];
	const errors: string[] = [];
	for (const field of ["cpu", "memory", "disk", "gpu"] as const) {
		const error = validateResourceQuantity(field, resources[field]);
		if (error) {
			errors.push(`${context} ${FIELD_LABEL[field]}: ${error}`);
		}
	}
	return errors;
}
