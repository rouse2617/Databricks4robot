# Task: Fix Frontend Bugs

Fix frontend issues found by code review.

## Issues to Fix

### 1. ComponentManager error swallowing
- **File:** `Frontend/src/components/pipeline/ComponentManager.tsx` + `Frontend/src/pages/PipelinePage.tsx`
- **Problem:** API save/delete failures are caught silently in parent callbacks, so the child never shows error states. localStorage diverges from backend.
- **Fix:** Don't catch in `handleComponentSave/Delete`, or rethrow after adding context. Show persistent error state in the component.

### 2. AssetPicker search race
- **File:** `Frontend/src/components/pipeline/AssetPicker.tsx`
- **Problem:** Search requests are not cancelled. A slow old response can overwrite a newer faster one.
- **Fix:** Use `AbortController` to cancel in-flight requests when new search fires.

### 3. AssetPicker error vs empty state
- **File:** `Frontend/src/components/pipeline/AssetPicker.tsx`
- **Problem:** Search errors are only toast'ed, leaving the UI showing "not found" — conflating API failure with no results.
- **Fix:** Track `error` state separately and show a retry/error UI.

### 4. lint failure
- **File:** `Frontend/src/pages/PipelinePage.tsx`
- **Fix:** Run `npx biome format --write src/pages/PipelinePage.tsx` to fix import ordering.

### 5. CSRF header on mutations
- **File:** `Frontend/src/api/pipelineClient.ts`
- **Fix:** Add `X-Requested-With: XMLHttpRequest` header to all POST/PUT/DELETE requests.

## Steps
1. Fix ComponentManager error handling
2. Fix AssetPicker race condition and error state
3. Fix lint
4. Add CSRF header
5. Run `npm run lint` to verify
6. Commit: `git add -A && git commit -m "fix: resolve frontend MAJOR and MINOR bugs"`
