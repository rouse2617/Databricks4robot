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

// Dynamically import @mcap/core — the worker has its own scope
let McapStreamReader: any;

interface McapWorkerMessage {
	type: "load" | "seek" | "close";
	data?: ArrayBuffer;
	url?: string;
	timestampNs?: number;
}

self.onmessage = async (e: MessageEvent<McapWorkerMessage>) => {
	const msg = e.data;

	if (msg.type === "load") {
		await loadMCAP(msg.data!, msg.url!);
	}
};

async function loadMCAP(data: ArrayBuffer, url: string) {
	try {
		// Dynamic import
		const mcap = await import("@mcap/core");
		McapStreamReader = mcap.McapStreamReader;

		const reader = new McapStreamReader({ decompressHandlers: {} });
		reader.append(new Uint8Array(data));

		const channels: Record<number, any> = {};
		const schemas: Record<number, any> = {};
		let minTime = Infinity;
		let maxTime = -Infinity;
		let videoCount = 0;

		// First pass: collect schema & channel info
		while (true) {
			const record = reader.nextRecord();
			if (!record) break;

			if (record.type === "Schema") {
				schemas[record.id] = record;
			} else if (record.type === "Channel") {
				channels[record.id] = record;
			} else if (record.type === "Message") {
				minTime = Math.min(minTime, Number(record.logTime));
				maxTime = Math.max(maxTime, Number(record.logTime));
				videoCount++;
			}
		}

		// Build channel list with video topics
		const channelList = Object.entries(channels)
			.filter(([_, ch]: [string, any]) =>
				ch.schemaName?.includes("CompressedVideo"),
			)
			.map(([id, ch]: [string, any]) => ({
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

			if (record.type === "Message") {
				const ch = channels[record.channelId];
				if (
					ch?.schemaName?.includes("CompressedVideo") &&
					!frameSamples[record.channelId]
				) {
					frameSamples[record.channelId] = true;
					self.postMessage({
						type: "frame",
						channelId: record.channelId,
						topic: ch.topic,
						data: record.data,
						timestampNs: Number(record.logTime),
					});
				}
			}
		}

		self.postMessage({ type: "done" });
	} catch (err: any) {
		self.postMessage({ type: "error", message: err.message || String(err) });
	}
}
