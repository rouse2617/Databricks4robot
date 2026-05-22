import { describe, expect, it } from "vitest";
import { formatRunIdShort, isRegisteredRunId } from "./runId";

describe("runId", () => {
	it("accepts 16-char alphanumeric", () => {
		expect(isRegisteredRunId("R001abc123def456")).toBe(true);
	});

	it("rejects legacy short ids", () => {
		expect(isRegisteredRunId("cyb1013-verify")).toBe(false);
		expect(isRegisteredRunId("run-1")).toBe(false);
	});

	it("formatRunIdShort truncates long ids", () => {
		expect(formatRunIdShort("R001abc123def456")).toBe("R001…f456");
	});
});
