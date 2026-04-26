# Workflow Management

Source: https://io-ai.tech/platform/en/guides/Modules/workflows/#related-features

- Feature Modules

- Workflow Management

# Workflow Management

When data volume is large and processing workflows are repetitive, manual operations are time-consuming and error-prone.

Typical scenarios:

- Newly uploaded MCAP files need automatic preprocessing

- Data from specific projects need automatic annotation task creation

- Completed annotation data need automatic quality checks

- HDF5 format data need automatic conversion to MCAP format

Workflow management is designed to solve these problems. By configuring matching rules and action rules, it can automatically execute data processing, annotation assignment, quality checks and other operations, improving efficiency and reducing manual intervention.

## Core Conceptsâ

### Matching Rulesâ

Define data matching conditions to identify datasets that need processing.

Rule Types:

- Match by Project: Match data from specific projects

- Match by Tag: Match data with specific tags

- Match by Robot Type: Match data collected by specific robots

- Match by Data Format: Match data in specific formats (such as MCAP, HDF5)

- Match by Status: Match data in specific status (such as unannotated, annotated)

Rule Configuration:

- Rule name and description

- Matching conditions (AND/OR logic)

- Rule priority

- Rule enable status

### Action Rulesâ

Define operations to execute on matched data.

Operation Types:

- Data Preprocessing: Automatic renaming, format conversion, etc.

- Automatic Annotation Assignment: Automatically create annotation tasks and assign

- Quality Check: Automatically execute quality checks

- Tag Addition: Automatically add tags

- Project Assignment: Automatically assign to projects

- Format Conversion: HDF5 â MCAP, LeRobot â MCAP, etc.

Rule Configuration:

- Rule name and description

- Execution action

- Action parameters

- Rule priority

### Workflowsâ

Combine matching rules and action rules into complete workflows.

Workflow Structure:

- Workflow name and description

- Associated matching rules

- Associated action rules

- Workflow priority

- Workflow enable status

Execution Flow:

- Data enters system or status changes

- Workflow engine checks matching rules

- If matched, execute associated action rules

- Record execution logs

## Quick Start: Create Your First Workflowâ

### Step 1: Create Matching Ruleâ

- Go to Workflow Management page, switch to "Matching Rules" tab

- Click "New Matching Rule"

- Fill in rule information:Rule Name: Clearly describe matching conditions (e.g., "Newly Uploaded MCAP Files")Rule Description: Additional notes (optional)

- Rule Name: Clearly describe matching conditions (e.g., "Newly Uploaded MCAP Files")

- Rule Description: Additional notes (optional)

- Configure matching conditions:Select matching fields (project, tag, format, etc.)Set matching valuesCan add multiple conditions, combine with AND/OR logic

- Select matching fields (project, tag, format, etc.)

- Set matching values

- Can add multiple conditions, combine with AND/OR logic

- Set priority and enable status

- Save rule

### Step 2: Create Action Ruleâ

- Switch to "Action Rules" tab

- Click "New Action Rule"

- Fill in rule information:Rule Name: Clearly describe executed operation (e.g., "Automatic Preprocessing")Rule Description: Additional notes (optional)

- Rule Name: Clearly describe executed operation (e.g., "Automatic Preprocessing")

- Rule Description: Additional notes (optional)

- Configure execution action:Select action type (data preprocessing, format conversion, etc.)Configure action parametersCan add multiple action steps

- Select action type (data preprocessing, format conversion, etc.)

- Configure action parameters

- Can add multiple action steps

- Set priority and enable status

- Save rule

### Step 3: Create Workflowâ

- Switch to "Workflows" tab

- Click "New Workflow"

- Fill in workflow information:Workflow Name: Clearly describe workflow purpose (e.g., "MCAP File Automatic Preprocessing")Workflow Description: Additional notes (optional)Project Scope: Select global or specific project

- Workflow Name: Clearly describe workflow purpose (e.g., "MCAP File Automatic Preprocessing")

- Workflow Description: Additional notes (optional)

- Project Scope: Select global or specific project

- Select rules:Select matching ruleSelect action rule

- Select matching rule

- Select action rule

- Set priority and enable status

- Save workflow

After workflow is created, the system will automatically monitor data status and automatically execute workflows when data matches rules.

## Advanced Usageâ

### How to Configure Complex Matching Conditions?â

Multi-Condition Combination:

Matching rules support combination of multiple conditions:

- AND Logic: All conditions must be satisfiedExample: Project = "Project A" AND Format = "MCAP" AND Status = "Unannotated"

AND Logic: All conditions must be satisfied

- Example: Project = "Project A" AND Format = "MCAP" AND Status = "Unannotated"

- OR Logic: Any condition satisfied is sufficientExample: Tag = "High Quality" OR Tag = "Test Data"

OR Logic: Any condition satisfied is sufficient

- Example: Tag = "High Quality" OR Tag = "Test Data"

Condition Types:

- Project Matching: Match data from specific projects

- Tag Matching: Match data with specific tags

- Robot Type Matching: Match data collected by specific robots

- Data Format Matching: Match data in specific formats

- Status Matching: Match data in specific status

### How to Configure Multiple Action Steps?â

Action Steps:

Action rules can contain multiple steps, executed in sequence:

- Automatic Renaming: Rename datasets according to rules

- Format Conversion: Convert data to standard format

- Automatic Project Import: Import data to specified project

- Run Custom Algorithm: Execute custom processing algorithms

Step Configuration:

Each step can configure independent parameters:

- Renaming rules: Set naming patterns and target formats

- Conversion parameters: Set conversion options and parameters

- Project selection: Select target project

### How to Set Workflow Priority?â

Priority Note:

- Smaller numbers mean higher priority

- When multiple workflows match, execute in priority order

- High priority workflows execute first

Setting Recommendations:

- Important workflows set higher priority (e.g., 1-10)

- General workflows set medium priority (e.g., 11-50)

- Optional workflows set lower priority (e.g., 51-100)

### How to Test Workflows?â

Rule Testing:

After creating rules, you can test if rules are correct:

- Click "Test" in rule list

- System will display list of matched datasets

- Confirm if matching results meet expectations

Workflow Testing:

- After creating workflow, can disable it first

- Manually trigger test, view execution results

- Enable workflow after confirming no issues

## Use Casesâ

### Scenario 1: Automatic Data Preprocessingâ

Requirement: Newly uploaded MCAP files need automatic preprocessing.

Configuration Steps:

- Create matching rule: Match newly uploaded MCAP filesCondition: Format = "MCAP" AND Status = "Newly Uploaded"

- Condition: Format = "MCAP" AND Status = "Newly Uploaded"

- Create action rule: Execute data preprocessing operationAction: Data Preprocessing

- Action: Data Preprocessing

- Create workflow: Associate matching rule and action rule

- Enable workflow: Newly uploaded MCAP files automatically trigger preprocessing

### Scenario 2: Automatic Annotation Assignmentâ

Requirement: Data from specific projects and unannotated need automatic annotation task creation.

Configuration Steps:

- Create matching rule: Match data from specific projects and unannotatedCondition: Project = "Project A" AND Status = "Unannotated"

- Condition: Project = "Project A" AND Status = "Unannotated"

- Create action rule: Automatically create annotation tasks and assign to annotatorsAction: Automatic Annotation AssignmentParameters: Specify annotators and reviewers

- Action: Automatic Annotation Assignment

- Parameters: Specify annotators and reviewers

- Create workflow: Associate matching rule and action rule

- Enable workflow: Data meeting conditions automatically create annotation tasks

### Scenario 3: Automatic Format Conversionâ

Requirement: HDF5 format data need automatic conversion to MCAP format.

Configuration Steps:

- Create matching rule: Match HDF5 format dataCondition: Format = "HDF5"

- Condition: Format = "HDF5"

- Create action rule: Execute format conversionAction: HDF5 â MCAP ConversionParameters: Select robot type (Agilex, Realman, etc.)

- Action: HDF5 â MCAP Conversion

- Parameters: Select robot type (Agilex, Realman, etc.)

- Create workflow: Associate matching rule and action rule

- Enable workflow: HDF5 data automatically convert to MCAP

## Workflow Managementâ

### How to View Workflow Execution Status?â

Execution Logs:

On the Workflow Management page you can view:

- Execution time

- Workflow name

- Matched datasets

- Executed actions

- Execution result (success/failure)

- Error information (if failed)

Query Functions:

- Filter by workflow

- Filter by time range

- Filter by execution status

- View execution details

### How to Manage Workflows?â

Workflow Operations:

- Enable/Disable: Control whether workflow executes

- Edit: Modify workflow configuration

- Delete: Delete unnecessary workflows

- Copy: Create new workflow based on existing workflow

Workflow List:

- Display list of all workflows

- Display workflow scope (project or global)

- Display enable status and priority

- Display creation and update time

## Common Questionsâ

### What to Do When Workflow Doesn't Execute?â

Possible Causes:

- Workflow Not Enabled: Check if workflow is enabled

- Matching Rule Doesn't Match: Check if matching rule is correct

- Priority Too Low: Other high priority workflows execute first

- Action Rule Error: Check if action rule configuration is correct

Solution:

- Check workflow enable status

- Test matching rule, confirm it can correctly match data

- View execution logs to understand failure reasons

- Adjust workflow priority

### How to Avoid Workflow Conflicts?â

Conflict Scenarios:

- Multiple workflows match same data

- Workflow execution order uncertain

- Action rules affect each other

Solution:

- Set Priority: Set clear priorities for workflows

- Refine Matching Conditions: Use more precise matching conditions to avoid overlap

- Test Verification: Test after creation, confirm no conflicts

### What to Do When Workflow Execution Fails?â

Handling Steps:

- View execution logs to understand failure reasons

- Check if action rule configuration is correct

- Check if data meets requirements

- Fix issues and re-execute

Common Errors:

- Data format doesn't meet requirements

- Action parameter configuration error

- Insufficient system resources

- Network or storage issues

## Applicable Rolesâ

### Administratorâ

You can:

- Create and manage all workflows

- Configure matching rules and action rules

- Monitor workflow execution status

- Optimize workflow performance

- Handle workflow exceptions

### Project Managerâ

You can:

- Create project-specific workflows

- Configure project-specific automation rules

- Monitor project workflow execution status

- Optimize project data processing workflows

## Related Featuresâ

After completing workflow management, you may also need:

- Data Management: View data processed by workflows

- Annotation Tasks: View automatically created annotation tasks

- System Settings: Configure data pipeline related settings