# Pipeline Spec Delta — CYB-1569 Pod Terminal Exec

## ADDED Requirements

### Requirement: Execution Target Terminal Policy

DataBrew SHALL treat Pod terminal exec as disabled unless the selected execution target explicitly enables it.

#### Scenario: terminal disabled by default
- **Given** a workflow run uses an execution target without terminal policy enabled
- **When** a user opens the node runtime tab
- **Then** the UI SHALL show that Pod terminal is disabled for this target
- **And** the backend SHALL reject terminal session creation with a non-credential-leaking forbidden/unavailable error.

#### Scenario: namespace isolation
- **Given** a workflow run belongs to namespace `N`
- **When** a terminal session is requested for a node Pod
- **Then** the backend SHALL only exec into a Pod resolved from that workflow/run in namespace `N`
- **And** client-provided pod, namespace, or cluster values SHALL NOT be trusted as authority.

### Requirement: Backend-Owned Exec Sessions

DataBrew SHALL create a backend-owned terminal session before opening any WebSocket stream to Kubernetes exec.

#### Scenario: create terminal session
- **Given** an allowed execution target, a running Pod node, and an allowed command preset
- **When** the user requests a terminal session
- **Then** the backend SHALL create a session record with run, workflow, node, pod, container, command, actor, target, namespace, expiry, and status
- **And** return attach metadata that does not expose Kubernetes credentials.

#### Scenario: attach terminal websocket when exec streaming is enabled
- **Given** a created terminal session with a valid attach token
- **And** backend Kubernetes exec streaming is enabled for the target
- **When** the browser attaches to the WebSocket
- **Then** the backend SHALL proxy terminal frames to Kubernetes exec
- **And** mark the session attached
- **And** reject expired, reused, or mismatched attach attempts.

#### Scenario: attach terminal websocket before exec streaming is enabled
- **Given** a created terminal session with a valid attach token
- **And** backend Kubernetes exec streaming is not enabled for the target
- **When** the browser attaches to the WebSocket
- **Then** the backend SHALL return controlled status, `POD_EXEC_UNAVAILABLE`, and exit frames
- **And** mark the session failed without exposing Kubernetes credentials.

### Requirement: Terminal Command Policy

DataBrew SHALL enforce command policy before creating a terminal session.

#### Scenario: allowed command
- **Given** target policy allows command `sh`
- **When** the user starts a terminal with command `sh`
- **Then** the backend SHALL allow the session if all other checks pass.

#### Scenario: denied command
- **Given** target policy denies `kubectl` or another dangerous command pattern
- **When** the user requests that command
- **Then** the backend SHALL reject the session before contacting Kubernetes
- **And** record the rejection as audit metadata.

### Requirement: Session Lifecycle And Audit

DataBrew SHALL record terminal session lifecycle in audit records and, when a run is known, in pipeline run events.

#### Scenario: session lifecycle events
- **Given** a terminal session is created, attached, and ended
- **When** the lifecycle transitions occur
- **Then** DataBrew SHALL write audit records for the actor/session/pod/command
- **And** append run events for session created, attached, and ended when the pipeline run exists.

#### Scenario: timeout
- **Given** a terminal session exceeds max duration or idle timeout
- **When** the timeout is reached
- **Then** the backend SHALL close the Kubernetes exec stream and WebSocket
- **And** mark the session timed out
- **And** record audit/run-event metadata.

### Requirement: Terminal Availability States

DataBrew SHALL make unavailable and forbidden terminal states explicit in the UI.

#### Scenario: completed Pod
- **Given** a node Pod is completed or deleted
- **When** the user opens the runtime tab
- **Then** the terminal action SHALL be disabled
- **And** the UI SHALL direct the user to logs, events, or diagnostics instead.

#### Scenario: Kubernetes unavailable
- **Given** the backend cannot reach the Kubernetes API
- **When** the user requests terminal capability
- **Then** the UI SHALL show Kubernetes terminal unavailable
- **And** no raw kubeconfig, token, or credential error SHALL be displayed.
