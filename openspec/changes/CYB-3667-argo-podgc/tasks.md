# Tasks

- [x] Set `PodGC: &wfv1.PodGC{Strategy: wfv1.PodGCOnWorkflowSuccess}` on the
      transpiled `WorkflowSpec` in `backend/internal/transpiler/transpiler.go`.
- [x] Add `TestTranspilePodGC` asserting the default strategy is
      `OnWorkflowSuccess`.
- [ ] Local gates (Tier M): `make fmt && make vet && go test ./internal/transpiler/...`.
- [ ] Deploy dev backend and verify on a live run:
      - submit a workflow that succeeds → step pods reclaimed shortly after
        completion (`kubectl get pods -n <ns> | grep <run-id>` empty).
      - a failing workflow → step pods retained (diagnosable).
      - `kubectl get workflow <uid> -o json` shows
        `spec.podGC.strategy: OnWorkflowSuccess`.
- [ ] Post-merge (optional): prune existing terminated pods in `video-proc-prod`.
