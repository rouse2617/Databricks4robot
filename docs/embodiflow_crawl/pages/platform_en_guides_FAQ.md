# FAQ

Source: https://io-ai.tech/platform/en/guides/FAQ/#contact

- FAQ

# FAQ

Common questions and solutions. If you canât find your answer, contact support.

## Account & Loginâ

### Q: Forgot password?â

A: Ask an admin to reset it. After 5 consecutive failures, the account and IP will be temporarily blocked.

### Q: Can I register a new account?â

A: Registration is controlled by admins. Ask an admin to create an account if needed.

### Q: Missing features after login?â

A: Likely insufficient permissions. Confirm your role/project access or ask an admin to adjust.

## Data Managementâ

### Q: How to upload data?â

A: On the "Upload" page, choose project and cloud storage. Video, audio, ROS data formats are supported.

### Q: What data formats are supported?â

A: Mcap, Bag, Mp4, audio, and other ROS-standard formats. Online transcoding to Mcap is supported. Video and audio files can be automatically converted to MCAP format in the browser.

### Q: Which browsers support video conversion?â

A: Video to MCAP conversion requires browser support for MediaStreamTrackProcessor API. Currently only the latest versions of Chrome (94+) and Edge (94+) browsers support this feature. Other browsers will display a clear error message.

### Q: Can audio files be uploaded?â

A: Yes. The platform supports audio formats including MP3, WAV, AAC, OGG, etc., and automatically converts them to MCAP format AudioData messages for convenient robot training.

### Q: Upload failed?â

A: Check file format, network stability, and cloud storage configuration. If uploading video files, ensure you're using Chrome or Edge browser. Failed tasks can be retried for upload.

### Q: What happens when re-uploading a deleted dataset?â

A: The system automatically detects if a soft-deleted dataset with the same name exists. If found, you can choose to recover the existing dataset (preserving all historical information) or create a new dataset.

## Annotation Tasksâ

### Q: How to create an annotation task?â

A: On the "Data" page, select data and click the bottom "Annotate" button, then fill in task details.

### Q: What task states exist?â

A: Including: To Start (PENDING), In Progress (IN_PROGRESS), To Review (PENDING_REVIEW), Approved (REVIEWED), Submitted (SUBMITTED), etc.

### Q: How to view detailed task information?â

A: Click the task name or ID in the task list to enter the task detail page. The detail page contains multiple tabs: Annotation Data, Batch Annotation, Invalid Annotations, Annotation Statistics, providing comprehensive task information.

### Q: Can tasks be batch deleted?â

A: Yes. Multi-select tasks in the task list, click the "Batch Delete" button at the bottom, and confirm to batch delete selected tasks. Delete operations cannot be recovered, please operate with caution.

### Q: What if data is returned by an auditor?â

A: Read the auditor's feedback, modify the annotation accordingly, and resubmit. You can view all invalid annotations in the "Invalid Annotations" tab on the task detail page.

## Project Managementâ

### Q: How to create a new project?â

A: On the "Projects" page, click "New Project" and set name and visibility.

### Q: How to assign tasks to team members?â

A: During task creation, select annotators and auditors.

### Q: How to view project progress?â

A: Use "Overview" to see overall progress and statistics.

## Cloud Storageâ

### Q: Which services are supported?â

A: Tencent COS, Aliyun OSS, Huawei OBS, Amazon S3, Azure Blob Storage, Cloudflare R2, MinIO, etc.

### Q: How to configure connections?â

A: On the "Cloud Storage" page, add a connection and fill in credentials.

### Q: Connection failed?â

A: Check network, credentials, and IP allowlist configuration.

## Systemâ

### Q: How to export annotated data?â

A: On the "Export" page, select data to export. Supports multiple formats including JSON, CSV, HDF5, LeRobot, MCAP, etc. Export tasks are automatically queued for processing, and you can view progress in the export history.

### Q: How to configure HDF5 export?â

A: HDF5 export requires setting two parameters: chunk size (number of original files each HDF5 file contains) and data refresh frequency (data samples per second, default 30Hz). See HDF5 export documentation for detailed instructions.

### Q: How to view export progress and history?â

A: In the "Export History" area of the data export page, you can view the list and status of all export tasks. Tasks in progress display real-time progress bars, and completed tasks can download result files.

### Q: How to view analytics charts?â

A: On the "Charts" page: action planning, durations, dependencies, calendar, etc.

### Q: How to manage dictionaries?â

A: On the "Dictionaries" page: skills, goals, objects, labels, etc.

## Mobileâ

### Q: Is mobile supported?â

A: Yes via web browsers. Desktop is recommended for best experience.

### Q: Any mobile limitations?â

A: Mainly viewing and simple operations. Use desktop for complex annotation.

## Supportâ

### Q: How to get help?â

A: 1) Read docs 2) Check module guides 3) Ask your admin 4) Contact support

### Q: How to report bugs or suggestions?â

A: Contact support or your admin. We value your feedback.

## Tipsâ

### Q: How to improve annotation efficiency?â

A:

- Learn shortcuts

- Use annotation templates

- Batch-process similar data

- Communicate regularly with the team

### Q: How to ensure annotation quality?â

A:

- Follow the annotation guideline carefully

- Use unified standards

- Do periodic quality checks

- Provide timely feedback and improvements

### Q: How to manage large-scale data?â

A:

- Use labels and categorization

- Clear naming conventions

- Regular cleanup and archiving

- Use bulk operations in the platform

## Model Training and Inferenceâ

### Q: What model training does the platform support?â

A: The platform supports various robot learning models, including: Vision-Language-Action models (SmolVLA, OpenVLA), Imitation Learning models (ACT, PI0, PI0Fast), Policy Learning models (Diffusion Policy, VQBET), Reinforcement Learning models (SAC, TDMPC), etc.

### Q: How to start model training?â

A: On the "Model Training" page, create a new training task, select data source (platform export data, URL, local upload, HuggingFace), configure model type, training parameters and computing resources, then click start training.

### Q: Can training be paused or stopped during training?â

A: Yes. On the training detail page, you can pause, resume, or stop training tasks. Pausing preserves current progress, resuming continues training. Stopping safely ends training and saves checkpoints.

### Q: How to deploy trained models?â

A: After training completion, on the training detail page select the checkpoint to deploy (recommend using "last" or "best"), click "Deploy Inference", configure service parameters, then deploy as inference service.

### Q: What testing methods do inference services support?â

A: Three testing methods are supported: Simulation Inference (using random data for quick validation), MCAP File Testing (using real data to verify inference effects), Offline Edge Deployment (deploy to robot local GPU).

### Q: What is task queue? How to manage it?â

A: Task queues are used to manage background tasks (such as data export, format conversion, etc.). Administrators can pause/resume queues, clear waiting queues, clean historical tasks, batch retry failed tasks, etc. See task queue management documentation for details.

## Contactâ

If you still need help, contact us:

- Support Email:support@io-ai.tech

- Support Phone: (+86) 0755-88658665

- Online Docs:Module Guides

Weâll get back to you as soon as possible.