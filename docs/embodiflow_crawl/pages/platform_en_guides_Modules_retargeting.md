# Action Retargeting

Source: https://io-ai.tech/platform/en/guides/Modules/retargeting/#related-features

- Feature Modules

- Action Retargeting

# Action Retargeting

Trained models are usually trained on specific robots. When you need to use them on another robot, you will encounter problems:

- Different Joint Counts: Humanoid robots have 20+ joints, while robotic arms may only have 6-7

- Different Motion Ranges: Different robots have very different joint motion ranges

- Different Kinematic Constraints: Each robot has its own kinematic limitations

Action retargeting is designed to solve this problem. It converts human motion capture data or actions from one robot into action sequences executable by another robot.

## Core Conceptsâ

Action retargeting involves three core databases:

### Human Motion Capture Trajectoriesâ

Human motion capture data, recording human joint motion trajectories. This data usually comes from motion capture systems (such as SenseXperience), containing complete human motion information.

Data Features:

- Multi-joint motion trajectories

- Time series data

- Support video preview

- Include action descriptions and tags

### Robot Reference Trajectoriesâ

Reference trajectories of robots executing specific actions, serving as targets for action retargeting. These trajectories record joint state sequences when robots execute actions.

Data Features:

- Robot joint state sequences

- Associated with specific robot models

- Support video demonstrations

- Can be associated with human motion capture trajectories

### Robot Skill Modelsâ

Robot skill models trained through action retargeting, can be directly used for robot control. Skill models map human actions to robot action space.

Data Features:

- Trained skill models

- Support video demonstrations

- Can be deployed to robots

- Include difficulty level and description

## Quick Start: From Human Demonstration to Robot Skillsâ

### Step 1: Upload Human Motion Capture Dataâ

- Go to the Action Retargeting page, switch to "Human Motion Capture Trajectories" tab

- Click "New", upload motion capture data file

- Upload video file (optional), for preview and demonstration

- Fill in action description and tags, set difficulty level

- After saving, data will appear in the list

Data Requirements:

- Support standard motion capture data formats

- Video files for visualization

- Recommend adding detailed descriptions for easy searching later

### Step 2: Create Robot Reference Trajectoriesâ

- Switch to "Robot Reference Trajectories" tab

- Select target robot model

- Click "New" to create reference trajectory

- Associate Human Motion Capture Trajectory: Select corresponding human motion capture data

- Upload Robot Demonstration Data: Upload reference data of robot executing this action

- Upload video file (optional), for demonstration

- After saving, reference trajectory will be associated with human motion capture trajectory

Why Do We Need Reference Trajectories?

Reference trajectories define how the target robot should execute this action. By comparing human motion capture and robot reference trajectories, the system can learn how to map human actions to robot action space.

### Step 3: Train Skill Modelâ

- Switch to "Robot Skill Models" tab

- Select corresponding robot reference trajectory

- Configure training parameters

- Start training

- After training completes, skill model can be directly used for robot control

Training Process:

The system will automatically:

- Analyze correspondence between human motion capture trajectories and robot reference trajectories

- Learn action mapping rules

- Consider robot kinematic constraints

- Generate skill models adapted to target robots

## Workflowâ

Complete workflow of action retargeting:

Workflow Steps:

- Upload Human Motion Capture Data: Upload human motion data collected by motion capture systems

- Create Robot Reference Trajectories: Create robot reference trajectories based on human motion capture, or upload existing robot demonstration data

- Action Mapping: Map human joints to robot joint space

- Trajectory Conversion: Convert action trajectories to adapt to robot kinematic constraints

- Skill Training: Train robot skill models to learn action mapping relationships

- Model Deployment: Deploy skill models to robots for actual control

## Advanced Usageâ

### How to Manage Three Databases?â

Human Motion Capture Trajectory Management:

- Upload Motion Capture Data: Support uploading motion capture data files and video files

- Add Descriptions and Tags: For easy searching and filtering later

- Set Difficulty Level: For classification management

- View Details: View complete trajectory information, video preview, associated reference trajectories and skill models

Robot Reference Trajectory Management:

- Filter by Robot: Select specific robot model, only show reference trajectories for that robot

- Associate Human Motion Capture: Associate reference trajectories with corresponding human motion capture trajectories

- Upload Reference Data: Upload reference data of robot executing actions

- View Details: View reference trajectory information, video demonstrations, associated human motion capture and skill models

Robot Skill Model Management:

- View Skill Library: Browse all trained skill models

- Filter by Difficulty: Filter skills by difficulty level

- Filter by Robot: View skill models for specific robots

- Model Deployment: Directly deploy skill models to robots

### Data Association Relationshipsâ

The platform maintains association relationships between three databases:

Human Motion Capture â Robot Reference Trajectories:

- One human motion capture trajectory can be associated with multiple robot reference trajectories

- Support reference trajectories for different robot models

- Association relationships used for action mapping and conversion

Robot Reference Trajectories â Robot Skill Models:

- One reference trajectory can generate multiple skill models

- Different training parameters can produce different skill models

- Skill models inherit action features from reference trajectories

Complete Chain:

- Human Motion Capture Trajectories â Robot Reference Trajectories â Robot Skill Models

- Form complete data chain from human demonstration to robot skills

- Support traceability and version management

### Search and Filterâ

Human Motion Capture Trajectories:

- Search by name

- Filter by difficulty level

- Filter by tags

- Sort by creation time

Robot Reference Trajectories:

- Filter by robot model

- Search by name

- Filter by associated human motion capture

- Sort by creation time

Robot Skill Models:

- Search by name

- Filter by difficulty level

- Filter by robot model

- Sort by creation time

## Use Casesâ

### Scenario 1: From Human Demonstration to Robot Skillsâ

Typical Process:

- Use motion capture systems to collect data of humans executing actions

- Upload to platform's human motion capture trajectory database

- Create corresponding robot reference trajectories

- Train robot skill models

- Deploy models to robots for actual control

Applicable Scenarios:

- Need to convert human actions to robot actions

- Have motion capture equipment and data

- Need to train skills for specific robots

### Scenario 2: Robot Demonstration Data Managementâ

Typical Process:

- Directly collect demonstration data on robots

- Upload to robot reference trajectory database

- Associate with corresponding human motion capture trajectories (optional)

- Train skill models

- Use for robot control

Applicable Scenarios:

- Already have robot demonstration data

- Need to manage and reuse demonstration data

- Need to train models based on demonstration data

### Scenario 3: Skill Model Reuseâ

Typical Process:

- View existing skill model library

- Select appropriate skill models

- Directly deploy to robots

- Or fine-tune based on existing models

Applicable Scenarios:

- Need to quickly use existing skills

- Multiple robots need the same skills

- Improve based on existing skills

## Common Questionsâ

### How to Choose Appropriate Reference Trajectories?â

Selection Recommendations:

- Robot Model Matching: Reference trajectories must match target robot model

- Action Similarity: Choose reference trajectories most similar to target actions

- Data Quality: Choose reference trajectories with high data quality

- Association Relationships: Prioritize reference trajectories already associated with human motion capture

### How Long Does It Take to Train Skill Models?â

Time Estimation:

Training time depends on:

- Data Volume: More data means longer training time

- Model Complexity: Complex models need more time

- Computing Resources: Better GPU performance means faster training

General Cases:

- Small-scale data: 10-30 minutes

- Medium-scale data: 30 minutes - 2 hours

- Large-scale data: More than 2 hours

### How to Verify Skill Model Effectiveness?â

Verification Methods:

- Video Demonstration: View video demonstrations of skill models

- Actual Testing: Test skill models on real robots

- Comparative Analysis: Compare effectiveness of different skill models

- User Feedback: Collect feedback from actual usage

### Which Robots Can Skill Models Be Used For?â

Usage Limitations:

- Skill models are usually bound to specific robot models

- Different robot models need retraining

- Robots with similar configurations may be reusable (needs testing)

## Applicable Rolesâ

### Administratorâ

You can:

- Manage all action retargeting databases

- Configure workflow rules

- Monitor data usage

- Ensure data quality and stable system operation

### Project Managerâ

You can:

- Manage project-related action retargeting data

- Coordinate researcher and developer work

- Track skill model training progress

- Ensure project goal achievement

### Researcherâ

You can:

- Upload motion capture data

- Create robot reference trajectories

- Train skill models

- Conduct experiments and verification

### Robot Developerâ

You can:

- Use trained skill models

- Deploy to robots for actual control

- Test model effectiveness

- Provide feedback for model improvement based on actual usage

## Related Featuresâ

After completing action retargeting, you may also need:

- Model Training: Train new skill models

- Inference Service: Deploy skill models for inference

- Robot Management: Manage robot device information