# Decisions — CYB-1535 Bounded Logs And Resources

## 2026-06-02 — No fake pagination over live logs
- **Context**: Live Argo/Kubernetes log APIs support tail and since filters but not stable offset pagination.
- **Decision**: Implement bounded tail and streaming first. Add cursor pagination only if persisted log chunks are introduced later.
- **Alternatives**: Present live logs as offset-paginated text.
- **Rationale**: Offset-style pagination over live logs would be unreliable and misleading for large or changing logs.

## 2026-06-02 — Proceed past OpenSpec checkpoint
- **Context**: Project default workflow says to stop after OpenSpec artifacts, but the user explicitly requested this worker implement CYB-1535 directly in the prepared worktree.
- **Decision**: Treat the user instruction as approval to continue runtime edits in this worktree without a separate checkpoint response.
- **Alternatives**: Stop after reading the already-created OpenSpec artifacts and ask for confirmation again.
- **Rationale**: Current-message user instruction has higher precedence and explicitly names the change, worktree, and implementation scope.
