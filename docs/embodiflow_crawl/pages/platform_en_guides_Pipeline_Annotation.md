# Data Annotation

Source: https://io-ai.tech/platform/en/guides/Pipeline/Annotation/#batch-annotation-workflow

- Data Pipeline

- Data Annotation

# Data Annotation

The core purpose of data annotation is to enable subsequent trained models to accurately understand the meaning of each action segment, so that the robot can correctly map human natural language instructions to specific actions. For example, when receiving an instruction like "clear the table", the robot can clearly identify the corresponding operation steps and involved items, achieving intelligent task execution.

### Annotation Flow Overviewâ

After selecting data on the Data page, you can follow either fine-grained or batch annotation; both paths end by saving to the server.

Platform supports two annotation methods:

- Fine-grained Annotation

- You can collect 10 minutes or even longer data and annotate each action time segment in this data.

- Supports grouped, nested, and overlapping annotations, marking both what actions were done and what things were achieved.

- Batch Annotation

- You can collect 100 pieces of data for the same action and annotate all these data with the same natural language at once.

- Supports automatic adaptation to tasks of different durations by time segments, such as marking what to do during the 10%-30% time segment.

## Data Viewingâ

Enter theDatapage and click on the corresponding file name link to start data viewing and annotation. The data page displays all available datasets, and you can quickly find the data you need through search and filtering functions.

### Panelsâ

A Panel is the rendering unit of data, equivalent to a camera perspective (Topic).

Each piece of data typically contains multiple camera perspectives, consisting of multiple panels. You can flexibly adjust the size, position, and number of panels to browse data in the best form. The flexible layout of panels allows you to view data from different angles simultaneously, significantly improving annotation efficiency.

### Switching Rendering Panelsâ

You can easily change the display content of any panel:

- Click the settings button in the upper left corner of the panel

- Select the desired panel type from the dropdown menu

This allows you to quickly switch between different data views according to current task requirements.

### Changing Subscription Messagesâ

You can also change the data source displayed by the panel:

- Click the settings button in the upper left corner of the panel

- Select the Topic message source you want to switch to in the popup settings menu

This allows you to view different data streams to meet the needs of various annotation scenarios.

## Fine-grained Annotationâ

The annotation method in this section is suitable for segmented annotation containing multiple atomic actions, especially suitable for scenarios where one piece of data contains multiple tasks. Fine-grained annotation allows you to mark multiple different actions or events in a single data stream.

For annotation task workflow, please refer to:

- Administrator creates annotation tasks

- Annotator executes annotation tasks

### Start Annotationâ

Click the "Start" button (or press shortcut keyQ) to mark the start time point of an action or event. The system will record the current timestamp as the starting point of the annotation.

You can:

- Auto play (press spacebar)

- Manually navigate forward and backward (press left and right arrow keys)

- Control navigation speed (press CTRL to accelerate or ALT to decelerate)

These flexible playback controls allow you to precisely locate the time points that need annotation.

### End Annotationâ

When the action or event ends, click the "End" button (or press shortcut keyR) to end the current annotation. The system will record the current timestamp as the end point of the annotation, thus determining the complete time range.

### Add Semantic Informationâ

After marking the time period, you need to add semantic information to the annotation:

- Fill in the form information

- Select the corresponding natural language description or add new dictionary items

- Save the information

Detailed semantic descriptions help with subsequent data training and analysis.

### Add Annotationâ

After completing the above steps, click the "Add Annotation" button (or pressEnter) to add this annotation to the annotation list of the current data. If you need to annotate multiple actions, repeat the above steps.

### Save Annotationâ

After completing all annotations, click the yellow upload button in the upper right corner to save the annotations to the server. Please make sure to save your work before leaving the page to prevent data loss.

## Batch Annotationâ

This function allows you to apply the annotation results of one piece of data to multiple homogeneous similar data, significantly saving annotation time.

For single task scenarios (such as 100 pieces of data all being the same action), the batch annotation function can greatly improve efficiency.

### Batch Annotation Workflowâ

- Select Sample Data: Batch select homogeneous data from the data list

Select Sample Data: Batch select homogeneous data from the data list

- Create Annotation Task: Create a new annotation task for this data

Create Annotation Task: Create a new annotation task for this data

- Start Annotation: View and start executing the annotation task

Start Annotation: View and start executing the annotation task

- Complete Sample Annotation: Complete the annotation of one sample according to the fine-grained annotation process

Complete Sample Annotation: Complete the annotation of one sample according to the fine-grained annotation process

- Save Sample: Save the completed sample annotation

Save Sample: Save the completed sample annotation

- Select Template: Return to the task page and select the annotated sample data as a template

Select Template: Return to the task page and select the annotated sample data as a template

- Batch Copy: Start batch copying annotations to other data

Batch Copy: Start batch copying annotations to other data

- Confirm Success: Confirm that the copy operation has been successfully completed

Confirm Success: Confirm that the copy operation has been successfully completed

- View Results: View the final results of batch annotation

View Results: View the final results of batch annotation

Through the batch annotation function, you can quickly apply standardized annotations to large amounts of data, especially suitable for processing large-scale homogeneous datasets, greatly improving work efficiency.

## Images on this page

- ![The dataset page displays all available data files](../images/platform_en_assets_images_dataset_79043558dcb1db594a34ea2a57fef7f6_webp.webp)
- ![Click the settings button in the upper left corner of the panel](../images/platform_en_assets_images_change_panel_267a0851a853d80cef32fef0dbbf0daf_webp.webp)
- ![Select the desired panel type from the dropdown menu](../images/platform_en_assets_images_change_panel2_1760044546f0518ddc6043d8af395deb_webp.webp)
- ![Change Topic message source through the settings menu](../images/platform_en_assets_images_change_topic_94196e8b6b6508794ea523ab65174d21_webp.webp)
- ![Preview the newly selected Topic content](../images/platform_en_assets_images_change_topic_preview_f34ffd1d4fae202822868e8434b8110e_webp.webp)
- ![Click the start button or use shortcut key Q to start annotation](../images/platform_en_assets_images_marker_start_5c60dd30641ded543f12cb15c1f265ee_webp.webp)
- ![Click the end button or use shortcut key R to end annotation](../images/platform_en_assets_images_marker_end_a9f922bec03e2f3903a89643507d2822_webp.webp)
- ![Fill in the natural language description of the annotation](../images/platform_en_assets_images_marker_form_4c1cb4c0ccab23cd6a6ae07bf7dcf2b4_webp.webp)
- ![Select predefined descriptions from options or add new descriptions](../images/platform_en_assets_images_marker_options_0ffc3b5d52d8aaf4237872b378ecbdb7_webp.webp)
- ![Click the add annotation button or use Enter key to confirm addition](../images/platform_en_assets_images_marker_add_f9e4fd1d66be6895d194c718dd1d34e5_webp.webp)
- ![Click the save button in the upper right corner to upload annotations to the server](../images/platform_en_assets_images_marker_save_b768c44721effa5d376674d9a2af5e77_webp.webp)
- ![Select one piece of data from the list as a sample](../images/platform_en_assets_images_select_data_adc7d25beea3fdc3e805c620937cae9f_webp.webp)
- ![Create a new annotation task](../images/platform_en_assets_images_create_task_42dc083bd5e4fd1961b010639b2db75d_webp.webp)
- ![View the created annotation task](../images/platform_en_assets_images_view_task_f2d05019be0f7ff6908756ea02a33d18_webp.webp)
- ![Complete the annotation of sample data](../images/platform_en_assets_images_annotation_sample_a5fd50f7c68d265a4e3a703aeabf5f76_webp.webp)
- ![Select the sample to use as a template](../images/platform_en_assets_images_select_sample_0b86e0c20fb3a7bbf8afd47eb4a094ed_webp.webp)
- ![Start the batch copy annotation process](../images/platform_en_assets_images_start_repeat_22d53a4804a81650bf283cb6dbd37031_webp.webp)
- ![System displays successful annotation copying](../images/platform_en_assets_images_repeat_success_054a6454665ffb1d4c6a0080d2f12540_webp.webp)
- ![View the final results of batch annotation](../images/platform_en_assets_images_repeat_result_49c7c4fe0f6bdb3ef0b9c411e17d37db_webp.webp)
