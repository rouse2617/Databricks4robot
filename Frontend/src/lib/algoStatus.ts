// Shared parsing/normalization for algo_results raw values.
// Used by both the algo matrix grid (AlgoProcessingPage) and the batch retry
// hook (useRetryAllFailed) so they always agree on which cells are "failed".

import type { CellStatus } from "../components/algo-matrix/AlgoStatusCell";

export type { CellStatus };

export function normalizeStatus(s: string): CellStatus {
  const lower = s.toLowerCase();
  if (lower === "ok" || lower === "success") return "ok";
  if (lower === "failed" || lower === "error") return "failed";
  if (lower === "running") return "running";
  if (lower === "pending") return "pending";
  if (lower === "blocked") return "blocked";
  return "none";
}

// Accepts either a plain status string ("ok", "failed", ...) or a JSON-encoded
// object with a `status` field. Returns the normalized CellStatus.
export function parseStatusFromRaw(raw?: string): CellStatus {
  if (!raw) return "none";
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === "object" && typeof parsed.status === "string") {
      return normalizeStatus(parsed.status);
    }
    return normalizeStatus(String(parsed));
  } catch {
    return normalizeStatus(raw);
  }
}

export function isFailedStatus(status: CellStatus): boolean {
  return status === "failed";
}
