# CYB-1676 — Pipeline Default Name

## Problem

The pipeline designer initializes every new blank canvas with the hard-coded name `my-pipeline`. Users repeatedly see the same value and can accidentally save multiple unrelated templates under an unclear default.

## Proposed Change

Generate a timestamp-based default name for new blank pipelines, while preserving explicit names loaded from saved templates, session edits, examples, and JSON imports.

## Non-Goals

- No backend API changes.
- No saved pipeline naming migration.
- No validation rule changes for pipeline names.

## Acceptance

- Opening a blank pipeline designer shows a generated name instead of `my-pipeline`.
- Importing or loading a saved pipeline still uses the pipeline's own name.
- Frontend tests cover the generated default name.
