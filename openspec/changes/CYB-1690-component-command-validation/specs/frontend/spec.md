# Frontend Spec Delta

## Modified Behavior

### Component command editor validates on change

When a user edits a component command in ComponentManager, the command input MUST parse valid JSON string arrays during change events and update the editable component state before blur.

#### Scenario: save valid command without blur
- **Given** the component editor is open
- **When** the user enters `["python", "src/main.py"]` and immediately saves
- **Then** the saved component command is `["python", "src/main.py"]`

#### Scenario: invalid command shows inline error
- **Given** the component editor is open
- **When** the user enters `python src/main.py`
- **Then** the command field shows an error state and a JSON format error message
- **And** the invalid value is not written into the editable command array

#### Scenario: non-string array shows inline error
- **Given** the component editor is open
- **When** the user enters `["python", 1]`
- **Then** the command field shows an error that the command must be a string array
