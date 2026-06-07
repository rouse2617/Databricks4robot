declare const __APP_VERSION__: string;
declare const __APP_BUILD_REF__: string;
declare const __APP_ENV__: string;

export interface AppVersionInfo {
	version: string;
	buildRef: string;
	shortBuildRef: string;
	environment: string;
}

function normalizeVersion(value: string | undefined): string {
	const normalized = value?.trim();
	return normalized || "unknown";
}

export function getShortBuildRef(buildRef: string): string {
	const normalized = normalizeVersion(buildRef);
	if (normalized === "unknown" || normalized === "local") return normalized;

	const [sha, ...suffixParts] = normalized.split("-");
	const shortSha = sha.length > 7 ? sha.slice(0, 7) : sha;
	return suffixParts.length > 0
		? `${shortSha}-${suffixParts.join("-")}`
		: shortSha;
}

export function getAppVersionInfo(): AppVersionInfo {
	const version = normalizeVersion(__APP_VERSION__);
	const buildRef = normalizeVersion(__APP_BUILD_REF__);
	return {
		version,
		buildRef,
		shortBuildRef: getShortBuildRef(buildRef),
		environment: normalizeVersion(__APP_ENV__ || import.meta.env.MODE),
	};
}

export function getAppVersionLabel(): string {
	const { version, shortBuildRef } = getAppVersionInfo();
	return `v${version} (${shortBuildRef})`;
}

export function getAppBuildBadgeLabel(): string {
	const { version, shortBuildRef, environment } = getAppVersionInfo();
	return `v${version} · ${shortBuildRef} · ${environment}`;
}
