/**
 * Low-volume write smoke: POST mcap-file then POST asset (unique IDs per iteration).
 * Intended for functional verification under load tooling, not sustained soak.
 *
 * Usage:
 *   BASE=http://HOST:8080 TOKEN=dev-token k6 run --vus 1 --iterations 3 k6-write-smoke.js
 */

import http from "k6/http";
import { check } from "k6";

const BASE = __ENV.BASE || "http://localhost:8080";
const TOKEN = __ENV.TOKEN || "dev-token";

function writeHeaders() {
	return {
		headers: {
			"X-Databrew-Token": TOKEN,
			"Content-Type": "application/json",
			"X-Request-ID": `k6-write-${__VU}-${__ITER}`,
		},
	};
}

function uuidV4() {
	return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
		const r = (Math.random() * 16) | 0;
		const v = c === "x" ? r : (r & 0x3) | 0x8;
		return v.toString(16);
	});
}

function hex32() {
	let s = "";
	for (let i = 0; i < 32; i++) {
		s += "0123456789abcdef"[(Math.random() * 16) | 0];
	}
	return s;
}

export default function () {
	const mcapId = uuidV4();
	const hash = hex32();

	const mcapPayload = JSON.stringify({
		mcap_file_id: mcapId,
		gcs_path: "gs://loadtest/path/file.mcap",
		raw_hash_md5: hash,
		ingest_state: "summarized",
		size_bytes: 1048576,
		file_duration_ms: 60000,
		start_timestamp_ns: 1700000000000000000,
		end_timestamp_ns: 1700000060000000000,
		channel_count: 12,
		chunk_count: 5,
		vendor_id: "loadtest",
		device_id: "k6-device",
		scene_id: "indoor",
		owner: "loadtest",
	});

	let r = http.post(`${BASE}/api/v1/mcap-files`, mcapPayload, writeHeaders());
	check(r, {
		"mcap POST 2xx": (res) => res.status >= 200 && res.status < 300,
	});

	const assetPayload = JSON.stringify({
		mcap_file_id: mcapId,
		start_timestamp_ns: 1700000000000000000,
		end_timestamp_ns: 1700000060000000000,
		reviewer: "k6",
		owner: "loadtest",
		type: "task_demo",
		tags: {
			priority: "low",
			quality: "good",
			scene: "indoor",
		},
	});

	r = http.post(`${BASE}/api/v1/assets`, assetPayload, writeHeaders());
	check(r, {
		"asset POST 201": (res) => res.status === 201,
	});
}
