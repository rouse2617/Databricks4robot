# Feature Modules

Source: https://io-ai.tech/platform/en/guides/Modules/#extensibility

- Feature Modules

# Feature Modules

The platform's capabilities are divided into two major categories:Data OperationsandGeneral Management, enabling efficient data processing and project management.

## Data Operationsâ

Covers the full workflow: data collection, upload, annotation, management, and analysis, supporting multi-role collaboration.

### Overviewâ

Access: Admin, Project Manager

Displays key indicators (dataset size, annotation duration, number of markers) and trend/distribution charts, helping you quickly grasp overall project health, with quick links to commonly used dictionaries.

Key features:

- Data statistics dashboard: Real-time metrics like total datasets, annotation duration, number of keypoints

- Quality monitoring: Distribution of annotation quality, including valid/invalid markers

- Project progress: Visualize data distribution and completion by project

- Recent activity: Track uploads and annotation activity in the last 7 days

- Common dictionaries: Quick access to frequently used labels and skill categories

### Dataâ

Access: Admin, Project Manager

Centralized data management and retrieval. Filter by name, robot, labels, etc. Supports batch operations (rename, statistics, annotation, labeling, deletion, import, robot association). Select data here to initiate annotation tasks with one click.

Key features:

- Dataset list: Filter/search by project, status, labels, etc.

- Data linking: Associate datasets with specific projects and robot devices

- Label management: Add custom labels for better categorization

- Data preview: Visualize Mcap data with keypoint views

- Batch operations: Bulk delete, rename, assign to projects

- Data QC (ROS recordings): Automatically checks frame rate, topics, time sync, and more against rules, withproject and global scope. After preprocessing, jobs are queued automatically (MCAPfirst today;bag/db3support rolling out). SeeQuality checks in the pipeline.

- Video QC: Multi-stream video diagnostics (frame drops, artifacts, BRISQUE), seeVideo QC.

### Uploadâ

Access: All roles

Upload video, audio, and ROS (BAG/MCAP) files and images to a selected project and cloud storage. Supports online transcoding to MCAP with progress and validation.

Key features:

- Multi-format support: Mcap, Bag, Mp4, audio, and other ROS standard formats

- Video conversion: Convert videos online to Mcap with quality, FPS, and audio settings

- Audio processing: Convert audio to Mcap for speech annotation

- Cloud integration: Multiple cloud backends with automatic sync

- Batch upload: Drag-and-drop and multi-file selection

- Progress monitoring: Real-time progress, speed, remaining time

- Format validation: Automatic format and integrity checks

### Collectionsâ

Access: Admin, Project Manager, Collector, Auditor

Create and manage data collection tasks, assign collectors/devices, and track execution. Collected data automatically flows into the platform.

Key features:

- Task creation: Define objectives, requirements, and device assignments

- Assignment: Assign tasks to devices or collectors

- Progress tracking: Monitor execution status in real time

- Device management: Manage teleoperation devices and upload settings

- Batch ops: Create, edit, delete tasks in bulk

### Annotation Tasksâ

Access: Admin, Project Manager, Annotator, Auditor

Manage all annotation tasks with status groups (To Start, In Progress, To Review, Approved, Submitted), with filtering and batch operations. To create a task, first select data on the "Data" page and click the bottom "Annotate" button.

Key features:

- Task management: Create, assign, and track annotation progress

- Status monitoring: Real-time stats for task states (Unassigned, In Progress, Completed, Reviewed)

- Batch operations: Bulk updates and assignments

- Filtering: Filter by project, status, creation time

- Access control: Different actions per role

- Progress stats: Visualize counts by state

### Dictionariesâ

Access: Admin, Project Manager, Annotator, Auditor

Maintain unified dictionaries for skills, objects, goals, adverbs, annotation labels, and data tags. Ensure consistent naming across projects with multilingual and batch operations.

Key features:

- Label categories: Manage multiple label types

- Multilingual: Support for Chinese, English, etc.

- Hierarchy: Hierarchical classification and ordering

- Batch ops: Import, edit, delete in bulk

- Usage stats: Label usage frequency tracking

- Versioning: Version history and management

### Chartsâ

Access: Admin, Project Manager

Provide analyses like action planning, relationships, duration, dependencies, and annotation calendar to reveal structure and patterns, detect bottlenecks and anomalies.

Key features:

- Sankey analysis: Behavior flows and transitions

- Duration stats: Bar charts for action durations and comparisons

- Dependencies: Analyze inter-action dependencies and order

- Calendar heatmap: Time-wise distribution and intensity

- Subtree analysis: Deep dive into subtask structures

- Real-time updates: Live refresh and dynamic charts

### Teleoperation Dataâ

Access: Admin, Project Manager

Designed for teleoperation products, supporting viewing and importing teleoperation data to improve data management efficiency.

Key features:

- Device management

- Data playback

- Connection monitoring

- Upload configuration

- Robot management

- Dataset management

### Importâ

Access: Admin

Integrated with local IO Agent software for bulk packaging, compression, and quality checks.

Key features:

- Local integration, bulk import, quality checks, format conversion, compression, progress monitoring

### Exportâ

Access: Admin

Export annotated data to JSON/CSV/HDF5 for further analysis and applications.

Key features:

- Multi-format export, selective export, batch export, history, quality report, custom parameters

### Skillsâ

Access: Admin, Project Manager

Manage robot skills: create, edit, categorize, and version control.

### Pluginsâ

Access: Admin, Project Manager

Extend platform capabilities via custom plugins: install, enable/disable, uninstall, and configure.

## General Managementâ

### Projectsâ

Access: Admin

Create, edit, and manage projects across their lifecycle.

### Usersâ

Access: Admin, Project Manager

Centralized user, permission, and project assignment management.

### Robotsâ

Access: Admin, Project Manager

Manage robot-related data for teleoperation products.

### Cloud Storageâ

Access: Admin

Connect and configure cloud storage backends (Tencent COS, Aliyun OSS, Huawei OBS, Amazon S3, Azure Blob Storage, Cloudflare R2, MinIO).

### Trashâ

Access: Admin

Centralized recycle bin for logical deletions with recovery options.

### Systemâ

Access: Admin

System-level configuration and management to ensure platform stability.

### Labsâ

Access: Admin, Project Manager

Experimental features for validating new capabilities.

### Sign Inâ

Access: All users

User authentication and login portal with multilingual UI.

## Technical Highlightsâ

### Internationalizationâ

- Chinese, English, Japanese

- Full i18n framework; all UI texts are translatable

- Users can switch the UI language

### Access Controlâ

- Role-based access control (RBAC)

- Fine-grained permissions

- Project-level data access control

### Real-timeâ

- WebSocket live updates

- Auto refresh and state sync

- Real-time progress and notifications

### Extensibilityâ

- Plugin architecture

- Open APIs for integration

- Modular design for customization

## Images on this page

- ![](../images/platform_en_assets_images_dashboard_abbfe03a569e141e712af6b90fe5dcc3_png.png)
- ![](../images/platform_en_assets_images_dataset_7852ff78f236a1118ad8aa1b22c37d09_png.png)
- ![](../images/platform_en_assets_images_upload_bc67b723cd8964311d64ccb24908afaa_png.png)
- ![](../images/platform_en_assets_images_collections_e6e0a949d09ff9c7cf8175efd895c03d_png.png)
- ![](../images/platform_en_assets_images_tasks_e6e0a949d09ff9c7cf8175efd895c03d_png.png)
- ![](../images/platform_en_assets_images_dicts_05f395363c4ee175a654b9be98b00fca_png.png)
- ![](../images/platform_en_assets_images_charts_3b19bd1583584cd73d63524a7e3c2a87_png.png)
- ![](../images/platform_en_assets_images_telexperience_dataset_020e6db74a08c8e23aedd6107b5965c3_png.png)
- ![](../images/platform_en_assets_images_import_b162185f5d070aaa7e33055156068888_png.png)
- ![](../images/platform_en_assets_images_export_f08f617bb632e45383881010a180f216_png.png)
- ![](../images/platform_en_assets_images_skills_dd06a7c2658d6bda0e4b2db8bc162b28_png.png)
- ![](../images/platform_en_assets_images_plugins_be2f3c38ad1fc27f83150951ee34370b_png.png)
- ![](../images/platform_en_assets_images_projects_c9b821d3a9b08b6fa7c5cdc6fbfc6342_png.png)
- ![](../images/platform_en_assets_images_users_c651a751304bf0e440c0bd3a0b955b3e_png.png)
- ![](../images/platform_en_assets_images_robots_55851247eebaf6c12588fbebe124e124_png.png)
- ![](../images/platform_en_assets_images_clouds_31c7345dde8f17ab0afaa74f53926ae1_png.png)
- ![](../images/platform_en_assets_images_trash_bbfde5142a23b1e709d9bc3416069d62_png.png)
- ![](../images/platform_en_assets_images_system_5fa99a5d4b6c9d190758ca042f8e0498_png.png)
- ![](../images/platform_en_assets_images_labs_e06f28ac7c175a55c4356161e5e10acf_png.png)
- ![](../images/platform_en_assets_images_signin_ac80489e3a3e68aba1a214bd7194c60a_png.png)
