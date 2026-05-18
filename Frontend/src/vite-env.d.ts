/// <reference types="vite/client" />

interface ImportMetaEnv {
	readonly VITE_API_BASE_URL?: string;
	readonly VITE_DEV_ACCESS_TOKEN?: string;
	readonly VITE_ENABLE_EMBED_LICHTBLICK_PLAYER?: string;
	/**
	 * Base URL for the standalone mcap-preview service. When empty (the
	 * default), the frontend issues same-origin requests and relies on
	 * the gateway/frontend proxy to dispatch /api/v1/preview/*.
	 * Override during local development to point at a port-forwarded
	 * preview pod, e.g. http://localhost:8090.
	 */
	readonly VITE_MCAP_PREVIEW_BASE_URL?: string;
}

interface ImportMeta {
	readonly env: ImportMetaEnv;
}
