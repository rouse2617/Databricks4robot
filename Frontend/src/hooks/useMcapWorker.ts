import { useCallback, useRef, useEffect, useState } from "react";

export interface ChannelInfo {
  id: number;
  topic: string;
  schemaName: string;
  messageEncoding: string;
}

export interface WorkerState {
  status: "idle" | "loading" | "parsing" | "ready" | "error";
  channels: ChannelInfo[];
  durationNs: number;
  messageCount: number;
  error?: string;
}

export function useMcapWorker() {
  const workerRef = useRef<Worker | null>(null);
  const [state, setState] = useState<WorkerState>({
    status: "idle",
    channels: [],
    durationNs: 0,
    messageCount: 0,
  });
  const frameCallbacks = useRef<
    Map<number, (data: Uint8Array, ts: number) => void>
  >(new Map());

  useEffect(() => {
    const worker = new Worker(
      new URL("../workers/mcapWorker.ts", import.meta.url),
      { type: "module" },
    );

    worker.onmessage = (e) => {
      const msg = e.data;
      switch (msg.type) {
        case "metadata":
          setState({
            status: "ready",
            channels: msg.channels,
            durationNs: msg.durationNs,
            messageCount: msg.messageCount,
          });
          break;
        case "frame":
          frameCallbacks.current.forEach((cb, chId) => {
            if (chId === msg.channelId) {
              cb(msg.data, msg.timestampNs);
            }
          });
          break;
        case "error":
          setState((s) => ({ ...s, status: "error", error: msg.message }));
          break;
        case "done":
          setState((s) => ({ ...s, status: "ready" }));
          break;
      }
    };

    workerRef.current = worker;
    return () => {
      worker.terminate();
    };
  }, []);

  const load = useCallback(async (url: string) => {
    setState((s) => ({ ...s, status: "loading" }));
    try {
      const res = await fetch(url);
      const data = await res.arrayBuffer();
      setState((s) => ({ ...s, status: "parsing" }));
      workerRef.current?.postMessage({ type: "load", data, url }, [data]);
    } catch (err: any) {
      setState((s) => ({ ...s, status: "error", error: err.message }));
    }
  }, []);

  const onFrame = useCallback(
    (channelId: number, cb: (data: Uint8Array, ts: number) => void) => {
      frameCallbacks.current.set(channelId, cb);
    },
    [],
  );

  return { state, load, onFrame };
}
