import { describe, expect, it } from "vitest";
import { isEmbedLichtblickPlayerEnabled } from "./embedLichtblickPlayer";

describe("isEmbedLichtblickPlayerEnabled", () => {
	it("enables via query param", () => {
		expect(isEmbedLichtblickPlayerEnabled({ search: "?embed_player=1" })).toBe(
			true,
		);
		expect(
			isEmbedLichtblickPlayerEnabled({ search: "?embedPlayer=true" }),
		).toBe(true);
		expect(isEmbedLichtblickPlayerEnabled({ search: "?lichtblick=yes" })).toBe(
			true,
		);
	});

	it("query param can explicitly disable", () => {
		expect(isEmbedLichtblickPlayerEnabled({ search: "?embed_player=0" })).toBe(
			false,
		);
		expect(
			isEmbedLichtblickPlayerEnabled({ search: "?lichtblick=false" }),
		).toBe(false);
	});

	it("falls back to env when query param absent", () => {
		expect(
			isEmbedLichtblickPlayerEnabled({
				search: "",
				env: { VITE_ENABLE_EMBED_LICHTBLICK_PLAYER: "true" } as ImportMetaEnv,
			}),
		).toBe(true);
		expect(
			isEmbedLichtblickPlayerEnabled({
				search: "",
				env: { VITE_ENABLE_EMBED_LICHTBLICK_PLAYER: "0" } as ImportMetaEnv,
			}),
		).toBe(false);
	});
});
