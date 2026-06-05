# CYB-1690 Component Command Validation

## Why

ComponentManager command input only syncs valid JSON into the editable component on blur and silently ignores invalid values. Saving directly after typing can persist a stale command, and invalid command text has no inline feedback.

## What Changes

- Parse command JSON on input change and update the editable command immediately when valid.
- Show inline validation errors for malformed JSON and non-string-array commands.
- Clear command validation state when loading a component into the editor.
- Add focused tests for save-without-blur and invalid command feedback.

## Impact

Frontend form behavior only. No pipeline contract, transpiler, or backend changes.
