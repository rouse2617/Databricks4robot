# pipeline Specification

## Purpose

Transpile and execute pipeline runs as Argo Workflows, including cost attribution for GKE-native billing tooling.

## Requirements

### Requirement: Pipeline run pods carry cost-tracking labels

The system SHALL attach labels identifying the batch job, template, and owner to every pod created for a submitted pipeline run, whenever the corresponding identifier is known at submission time — including for backfill batch items, whose owner is sourced from the owning job's creator.

### Requirement: Cost-tracking label values remain valid Kubernetes labels

The system SHALL sanitize any identifier before using it as a label value, so that no pipeline run submission fails or is rejected by Kubernetes due to an invalid label value.
