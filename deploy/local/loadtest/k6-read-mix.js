/**
 * Read-heavy mix against the full-stack backend (after `docker compose --profile full`).
 *
 * Usage:
 *   BASE=http://HOST:8080 TOKEN=dev-token VUS=15 k6 run k6-read-mix.js
 *   # Quick smoke (short plateau):
 *   SOAK_DURATION=45s RAMP_UP=15s RAMP_DOWN=15s VUS=10 k6 run k6-read-mix.js
 *
 * Env:
 *   BASE           — default http://localhost:8080
 *   TOKEN          — X-Grace-Token (default dev-token)
 *   VUS            — peak virtual users (default 10, max 200)
 *   RAMP_UP        — ramp-up stage duration (default 30s)
 *   SOAK_DURATION  — steady soak at VUS (default 10m)
 *   RAMP_DOWN      — ramp-down to 0 VUs (default 60s)
 */

import http from "k6/http";
import { check, sleep } from "k6";

const BASE = __ENV.BASE || "http://localhost:8080";
const TOKEN = __ENV.TOKEN || "dev-token";
const VUS = Math.min(200, Math.max(1, parseInt(__ENV.VUS || "10", 10)));
const RAMP_UP = __ENV.RAMP_UP || "30s";
const SOAK_DURATION = __ENV.SOAK_DURATION || "10m";
const RAMP_DOWN = __ENV.RAMP_DOWN || "60s";

function authHeaders() {
	return {
		headers: {
			"X-Grace-Token": TOKEN,
			"X-Request-ID": `k6-${__VU}-${__ITER}`,
		},
	};
}

export const options = {
	scenarios: {
		ramp_reads: {
			executor: "ramping-vus",
			startVUs: 0,
			stages: [
				{ duration: RAMP_UP, target: VUS },
				{ duration: SOAK_DURATION, target: VUS },
				{ duration: RAMP_DOWN, target: 0 },
			],
			gracefulRampDown: "60s",
			exec: "readMix",
		},
	},
	thresholds: {
		http_req_failed: ["rate<0.15"],
		http_req_duration: ["p(95)<8000"],
	},
};

export function readMix() {
	let r = http.get(`${BASE}/healthz`);
	check(r, { "healthz 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/assets?page=1&page_size=10`, authHeaders());
	check(r, { "assets list 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/search/assets?page=1&page_size=5`, authHeaders());
	check(r, { "search 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/lakehouse/status`, authHeaders());
	check(r, { "lakehouse status 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/lakehouse/tables`, authHeaders());
	check(r, { "lakehouse tables 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/algo-registry`, authHeaders());
	check(r, { "algo-registry 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/tag-registry`, authHeaders());
	check(r, { "tag-registry 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/deliveries?page=1&page_size=5`, authHeaders());
	check(r, { "deliveries 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/mcap-files?page=1&page_size=5`, authHeaders());
	check(r, { "mcap-files 200": (res) => res.status === 200 });

	r = http.get(`${BASE}/api/v1/metrics/registry`, authHeaders());
	check(r, { "metrics registry 200": (res) => res.status === 200 });

	sleep(0.2 + Math.random() * 0.3);
}
