declare const __APP_VERSION__: string;
declare const __APP_BUILD_REF__: string;

export function getAppVersionInfo(): { version: string; buildRef: string } {
	const version = __APP_VERSION__ || "unknown";
	const buildRef = __APP_BUILD_REF__ || "unknown";
	return { version, buildRef };
}

export function getAppVersionLabel(): string {
	const { version, buildRef } = getAppVersionInfo();
	return `v${version} (${buildRef})`;
}
