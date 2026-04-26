# Data Annotation

Source: https://io-ai.tech/platform/en/guides/Annotation/#batch-annotation-process

- Data Annotation

# Data Annotation

## Data Viewingâ

Go to theDatapage and click the corresponding file name link to start viewing and annotating data. The data page displays all available datasets, and you can quickly find the data you need using the search and filter functions.

### Panelâ

A panel is the rendering unit of data, similar to a camera perspective (Topic).

Each data entry usually contains multiple camera perspectives, which are represented by multiple panels. You can flexibly adjust the size, position, and number of panels to browse the data in the optimal way. The flexible layout allows you to view data from different angles simultaneously, significantly improving annotation efficiency.

### Switching Render Panelsâ

You can easily change the content displayed in any panel:

- Click the settings button in the upper left corner of the panel

- Select the desired panel type from the dropdown menu

This allows you to quickly switch between different data views according to your current task needs.

### Changing Subscription Topicsâ

You can also change the data source displayed in the panel:

- Click the settings button in the upper left corner of the panel

- In the pop-up settings menu, select the Topic message source you want to switch to

This enables you to view different data streams to meet various annotation scenarios.

## Fine-grained Annotationâ

The annotation method in this section is suitable for segment annotation containing multiple atomic actions, especially when a single data entry contains multiple tasks. Fine-grained annotation allows you to mark multiple different actions or events within a single data stream.

For annotation task workflow, please refer to:

- Manager: Create Annotation Task

- Annotator: Execute Annotation Task

### Start Annotationâ

Click the "Start" button (or press shortcut keyQ) to mark the start time point of an action or event. The system will record the current timestamp as the start point of the annotation.

You can:

- Play automatically (press Space)

- Manually jump forward/backward (press Left/Right arrow keys)

- Control jump speed (hold CTRL to speed up or ALT to slow down)

These flexible playback controls help you precisely locate the time point to annotate.

### End Annotationâ

When the action or event ends, click the "End" button (or press shortcut keyR) to end the current annotation. The system will record the current timestamp as the end point, thus determining the complete time range.

### Add Semantic Informationâ

After marking the time segment, you need to add semantic information for the annotation:

- Fill in the form information

- Select the corresponding natural language description or add a new dictionary entry

- Save the information

Detailed semantic descriptions help with subsequent data training and analysis.

### Add Annotationâ

After completing the above steps, click the "Add Annotation" button (or pressEnter) to add this annotation to the current data's annotation list. If you need to annotate multiple actions, repeat the above steps.

### Save Annotationâ

After completing all annotations, click the yellow upload button in the upper right corner to save the annotations to the server. Please make sure to save your work before leaving the page to prevent data loss.

## Batch Annotationâ

This feature allows you to apply the annotation results of one data entry to multiple similar homogeneous data entries, significantly saving annotation time.

For single-task scenarios (such as 100 data entries with the same action), the batch annotation feature can greatly improve efficiency.

### Batch Annotation Processâ

- Select Sample Data: Batch select homogeneous data from the data list

Select Sample Data: Batch select homogeneous data from the data list

- Create Annotation Task: Create a new annotation task for these data entries

Create Annotation Task: Create a new annotation task for these data entries

- Start Annotation: View and start executing the annotation task

Start Annotation: View and start executing the annotation task

- Complete Sample Annotation: Complete the annotation of a sample according to the fine-grained annotation process

Complete Sample Annotation: Complete the annotation of a sample according to the fine-grained annotation process

- Save Sample: Save the completed sample annotation

Save Sample: Save the completed sample annotation

- Select Template: Return to the task page and select the annotated sample data as a template

Select Template: Return to the task page and select the annotated sample data as a template

- Batch Copy: Start batch copying the annotation to other data entries

Batch Copy: Start batch copying the annotation to other data entries

- Confirm Success: Confirm that the copy operation was successful

Confirm Success: Confirm that the copy operation was successful

- View Results: View the final results of the batch annotation

View Results: View the final results of the batch annotation

With the batch annotation feature, you can quickly apply standardized annotations to large amounts of data, which is especially suitable for handling large-scale homogeneous datasets and greatly improves work efficiency.

## Images on this page

- ![The dataset page displays all available data files](../images/platform_en_assets_images_dataset_79043558dcb1db594a34ea2a57fef7f6_webp.webp)
- ![Click the settings button in the upper left corner of the panel](../images/platform_en_assets_images_change_panel_267a0851a853d80cef32fef0dbbf0daf_webp.webp)
- ![Select the desired panel type from the dropdown menu](../images/platform_en_assets_images_change_panel2_1760044546f0518ddc6043d8af395deb_webp.webp)
- ![Change the Topic message source through the settings menu](../images/platform_en_assets_images_change_topic_94196e8b6b6508794ea523ab65174d21_webp.webp)
- ![Preview the newly selected Topic content](../images/platform_en_assets_images_change_topic_preview_f34ffd1d4fae202822868e8434b8110e_webp.webp)
- ![Click the start button or use shortcut Q to start annotation](../images/platform_en_assets_images_marker_start_5c60dd30641ded543f12cb15c1f265ee_webp.webp)
- ![Click the end button or use shortcut R to end annotation](../images/platform_en_assets_images_marker_end_a9f922bec03e2f3903a89643507d2822_webp.webp)
- ![Fill in the natural language description for the annotation](../images/platform_en_assets_images_marker_form_4c1cb4c0ccab23cd6a6ae07bf7dcf2b4_webp.webp)
- ![Select a predefined description or add a new one](../images/platform_en_assets_images_marker_options_0ffc3b5d52d8aaf4237872b378ecbdb7_webp.webp)
- ![Click the add annotation button or use Enter to confirm](../images/platform_en_assets_images_marker_add_f9e4fd1d66be6895d194c718dd1d34e5_webp.webp)
- ![Click the save button in the upper right corner to upload annotations to the server](../images/platform_en_assets_images_marker_save_b768c44721effa5d376674d9a2af5e77_webp.webp)
- ![Select a data entry as a sample from the list](../images/platform_en_assets_images_select_data_adc7d25beea3fdc3e805c620937cae9f_webp.webp)
- ![Create a new annotation task](../images/platform_en_assets_images_create_task_42dc083bd5e4fd1961b010639b2db75d_webp.webp)
- ![View the created annotation task](../images/platform_en_assets_images_view_task_f2d05019be0f7ff6908756ea02a33d18_webp.webp)
- ![Complete the annotation of the sample data](../images/platform_en_assets_images_annotation_sample_a5fd50f7c68d265a4e3a703aeabf5f76_webp.webp)
- ![Select the sample to use as a template](../images/platform_en_assets_images_select_sample_0b86e0c20fb3a7bbf8afd47eb4a094ed_webp.webp)
- ![Start the batch copy annotation process](../images/platform_en_assets_images_start_repeat_22d53a4804a81650bf283cb6dbd37031_webp.webp)
- ![System shows annotation copy succeeded](../images/platform_en_assets_images_repeat_success_054a6454665ffb1d4c6a0080d2f12540_webp.webp)
- ![View the final results of the batch annotation](../images/platform_en_assets_images_repeat_result_49c7c4fe0f6bdb3ef0b9c411e17d37db_webp.webp)
