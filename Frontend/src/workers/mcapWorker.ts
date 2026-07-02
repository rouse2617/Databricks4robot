/**
 * WebWorker for parsing MCAP files in a background thread.
 *
 * Messages from main thread:
 *   { type: "load", data: ArrayBuffer, url: string }
 *   { type: "seek", timestampNs: number }
 *
 * Messages to main thread:
 *   { type: "metadata", channels, schemas, durationNs }
 *   { type: "frame", topic, data: Uint8Array, timestampNs, width, height }
 *   { type: "error", message: string }
 *   { type: "progress", loaded: number, total: number }
 */

// Dynamically import @mcap/core — the worker has its own scope.
// @mcap/core's TypeScript types don't expose the discriminator we need
// (`record.type`), so we treat records as `unknown` and narrow inline.
type McapRecord = {
	type?: "Schema" | "Channel" | "Message" | string;
	id?: number;
	name?: string;
	encoding?: string;
	schemaId?: number;
	channelId?: number;
	topic?: string;
	messageEncoding?: string;
	schemaName?: string;
	logTime?: bigint;
	data?: Uint8Array;
};
type McapStreamReaderCtor = new (opts: {
	decompressHandlers: Record<string, unknown>;
}) => {
	append(data: Uint8Array): void;
	nextRecord(): McapRecord | null;
};
let McapStreamReader: McapStreamReaderCtor | null = null;

interface McapWorkerMessage {
	type: "load" | "seek" | "close";
	data?: ArrayBuffer;
	url?: string;
	timestampNs?: number;
}

self.onmessage = async (e: MessageEvent<McapWorkerMessage>) => {
	const msg = e.data;

	if (msg.type === "load" && msg.data && msg.url) {
		await loadMCAP(msg.data, msg.url);
	}
};

async function loadMCAP(data: ArrayBuffer, url: string) {
	try {
		// Dynamic import
		const mcap = await import("@mcap/core");
		McapStreamReader = mcap.McapStreamReader as unknown as McapStreamReaderCtor;

		const reader = new McapStreamReader({ decompressHandlers: {} });
		reader.append(new Uint8Array(data));

		const channels: Record<
			number,
			{ schemaName?: string; topic?: string; messageEncoding?: string }
		> = {};
		const schemas: Record<number, { name?: string; encoding?: string }> = {};
		let minTime = Infinity;
		let maxTime = -Infinity;
		let videoCount = 0;

		// First pass: collect schema & channel info
		while (true) {
			const record = reader.nextRecord();
			if (!record) break;

			if (record.type === "Schema" && record.id != null) {
				schemas[record.id] = record;
			} else if (record.type === "Channel" && record.id != null) {
				channels[record.id] = record;
			} else if (record.type === "Message") {
				minTime = Math.min(minTime, Number(record.logTime));
				maxTime = Math.max(maxTime, Number(record.logTime));
				videoCount++;
			}
		}

		// Build channel list with video topics
		const channelList = Object.entries(channels)
			.filter(([_id, ch]) => ch.schemaName?.includes("CompressedVideo"))
			.map(([id, ch]) => ({
				id: Number(id),
				topic: ch.topic,
				schemaName: ch.schemaName,
				messageEncoding: ch.messageEncoding,
			}));

		self.postMessage({
			type: "metadata",
			channels: channelList,
			durationNs: maxTime - minTime,
			messageCount: videoCount,
			startNs: minTime,
			endNs: maxTime,
			url,
		});

		// Second pass: extract video frames (sample first frame from each channel)
		const reader2 = new McapStreamReader({ decompressHandlers: {} });
		reader2.append(new Uint8Array(data));

		const frameSamples: Record<number, boolean> = {};

		while (true) {
			const record = reader2.nextRecord();
			if (!record) break;

			if (record.type === "Message" && record.channelId != null) {
				const channelId = record.channelId;
				const ch = channels[channelId];
				if (
					ch?.schemaName?.includes("CompressedVideo") &&
					!frameSamples[channelId]
				) {
					frameSamples[channelId] = true;
					self.postMessage({
						type: "frame",
						channelId,
						topic: ch.topic,
						data: record.data,
						timestampNs: Number(record.logTime),
					});
				}
			}
		}

		self.postMessage({ type: "done" });
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);
		self.postMessage({ type: "error", message });
	}
}
