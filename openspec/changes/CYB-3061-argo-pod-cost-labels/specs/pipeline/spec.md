## ADDED Requirements

### Requirement: Pipeline run pods carry cost-tracking labels
The system SHALL attach labels identifying the batch job, template, and owner to every pod created for a submitted pipeline run, whenever the corresponding identifier is known at submission time.

**Priority**: P1 (High)
**Rationale**: GKE Cost Allocation is already enabled on the cluster; without business-domain labels on pods, real GCP billing data can only be sliced by raw workflow name, forcing a manual join back to the application database for every cost query. Labeling at submission time removes that join permanently, for every run going forward.

#### Scenario: Batch run submitted with a known batch job, template, and owner
- **Given** a user submits a pipeline run that belongs to a batch job, with a known template and an authenticated owner
- **When** the run is transpiled into an Argo Workflow and submitted
- **Then** every pod created for that workflow carries a label identifying the batch job, a label identifying the template, and a label identifying the owner

#### Scenario: Ad-hoc run submitted with no batch association
- **Given** a user submits a one-off pipeline run that is not part of any batch job
- **When** the run is transpiled into an Argo Workflow and submitted
- **Then** the resulting pods carry no batch-job label, while still carrying whichever of the template/owner labels are known — omitting an unknown identifier rather than emitting an empty-valued label

### Requirement: Cost-tracking label values remain valid Kubernetes labels
The system SHALL sanitize any identifier before using it as a label value, so that no pipeline run submission fails or is rejected by Kubernetes due to an invalid label value.

**Priority**: P0 (Critical)
**Rationale**: Kubernetes rejects pod creation outright if a label value violates its charset/length rules; an owner identifier is an email address and contains `@`, which is illegal in a label value. Without sanitization, adding these labels would break every run submission for every owner, not just fail to add the label.

#### Scenario: Owner identifier is an email address
- **Given** the owner of a pipeline run is identified by an email address containing `@` and `.`
- **When** the system builds the owner label for that run's pods
- **Then** the resulting label value contains no illegal characters and the workflow is accepted by Kubernetes

#### Scenario: Identifier exceeds the Kubernetes label value length limit or contains unexpected characters
- **Given** an identifier value that is longer than 63 characters, or contains characters outside the Kubernetes-legal label charset
- **When** the system builds the corresponding label for that run's pods
- **Then** the resulting label value is truncated to the legal length and stripped/escaped to only legal characters, so workflow submission still succeeds
