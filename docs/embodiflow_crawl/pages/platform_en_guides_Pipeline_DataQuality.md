# Quality checks

Source: https://io-ai.tech/platform/en/guides/Pipeline/DataQuality/#see-also

- Data Pipeline

- Quality checks

# Quality checks

When training models, issues like low frame rate, missing sensor topics, or poor time alignment often surface halfway through trainingâwasting compute and making root cause hard.Quality checksrunafter preprocessingandbefore export or training: a configurable standard scans eachROS recording(e.g..mcap,.bag,.db3) and marks itpassorfail.

.mcap

.bag

.db3

The platform turns this into a product feature. Admins and project managers configure rules in the QC UI; the system queues scans and shows results on the dataset list and detail pagesâno custom scan scripts required.

SeeData QCfor priorities, dataset name globs, and how this differs from video QC. This page focuseswhere QC sits in the end-to-end pipeline.

## Quick startâ

### When QC runsâ

Data must bepreprocessedinto aQC-ready ROS recording formatbefore QC applies. After preprocessing, if your rules match, the systemqueues scans automatically; you can alsorun again manuallyfrom a datasetâs detail page.

### Rule scopeâ

- Project rules: Apply only to datasets in the project you pickâgood for per-project standards.

- Global rules: Apply toall ROS recordingson the platform (including datasets not yet in a project)âgood for org-wide baselines.

The same dataset can match multiple rules; each rule is evaluated independently.

### Human review of resultsâ

When the machine marks a run as failed,admins or project managers(with permission) canoverridea result (e.g. confirm a false positive). Lists, tags, and export behavior use theeffective verdictâmanual override wins over the machine. Admins can also turn on policies such asblock export when QC failsto tie QC to export.

## Where this sits in the pipelineâ

From upload to training, the flow is roughly below. QC sits right after preprocessing to filter clearly bad data before filtering, export, and training.

## What you can configure in rulesâ

In the rule editor you typically see three kinds of settings (labels follow the UI):

Numeric thresholdsPick a metric (e.g. frame rate, duration, multi-stream time alignment), a comparator (â¥, â¤, etc.), and a threshold. Use for whole-file or per-sensor baselines.

Topic presencee.g. âmust include/joint_statesâ or âmust not include a debug topicâ. Use to ensure critical sensors were recorded.

/joint_states

Severity: blocking vs warning

- Blocking (fail): If this condition fails, thewhole QC runis marked failed.

- Warning: Shown in detail for awareness;a warning alone does not fail the whole runâgood to observe distributions before tightening.

Usuallyall conditions in one rulemust pass for that run to pass. For new rules, start withwarnings, then switch critical items toblockingwhen you are ready to enforce them.

After you create, enable, or edit a rule, the platformre-scans historical datathat already matched that rule so results match the latest definition. This runs in the backgroundâyou do not need to watch the page.

## Common metrics (for the UI)â

The two tables mirror names in the UI and help you choose upper vs lower bounds. Forhigher is better, useâ¥as a floor; forlower is better, useâ¤as a ceiling.

### Whole file / main streamâ

Metric

Meaning

Unit

Direction

Typical use

Recording duration

Time from first to last message in the file

s

Higher better

Drop too-short clips

Timestamp regressions

How often time âgoes backwardâ

count

Lower better (0 ideal)

Timeline anomalies

Cross-topic sync (P95 / P99 / max)

Multi-sensor time alignment error

ms

Lower better

Sync SLAs

Reference frame rate

Main stream average message rate

Hz

Higher better

Minimum FPS

Frame gaps (median / P95 / P99 / max)

Time between adjacent framesâjitter & stalls

ms

Lower better

Rhythm & freezes

Drop-frame count

Segments much longer than normal rhythm

count

Lower better

Gaps / packet loss

Sharpness (high percentile)

Image sharpness score

score

Higher better

Blur / focus

Exposure outlier ratio

Share of frames with abnormal brightness

ratio

Lower better

Unstable exposure

### Per topic or per typeâ

These are computedfor each matching topic. Scope must be âby topic nameâ or âby message typeâ; globs are supported. Ifanymatched row fails, that assertion fails.

Metric

Meaning

Unit

Direction

Typical use

Per-topic message rate

Approximate Hz for that stream

Hz

Higher better

Minimum camera FPS

Per-topic max frame gap

Worst pause on that stream

ms

Lower better

Worst single-stream stall

Per-topic message count

Number of messages

count

Higher better

Avoid âalmost emptyâ streams

Per-topic span

Time from first to last message on that topic

s

Higher better

Mid-recording dropouts

Per-topic first / last time

Position on the file timeline

s

Task-dependent

Advanced

## Example setups (tune numbers on site)â

- Org baseline: Global ruleâmain rate â¥ 15 Hz, duration â¥ 5 s, severity blocking.

- Clean timeline: Timestamp regressions â¤ 0.

- Multi-sensor sync: Cross-topic sync P99 â¤ 100 ms; usemaxif you care about spikes.

- Must-have topics: âRequired topicâ with real names (e.g./joint_states).

/joint_states

- Multi-camera: Scope matches image topics; per stream rate â¥ 10 Hz and max frame gap â¤ 500 ms.

- Overall sharpness: Scope âallâ; sharpness high percentile â¥ 40 (calibrate yourself).

- Drop frames: Drop count â¤ 10; usewarningif you only want visibility, not blocking export.

- No debug streams: âForbidden topicâ for streams that must not enter training bundles.

## Suggested workflowâ

- Maintain rules on theData QCpage (project or global; dataset name globs as needed).

- Check summaries on the dataset list or detail; open QC history for each runâs detail.

- For false positives, override with a clear reason; for real defects, fix upstream or re-preprocess and re-run.

- Export:Data export; training:Model training.

Admins or project managers can override a case and set the effective verdict to pass.

## See alsoâ

- Data QC

- Data export

- Model training

## Images on this page

- ![Quality checks in the platform flow](../images/platform_en_assets_images_qc_dataset_overview_bf60adfe833c303f0c9e778afd89277e_webp.webp)
- ![Add a QC rule](../images/platform_en_assets_images_qc_add_rule_5df0da546ccb05abfe1603fc34406733_webp.webp)
- ![QC summary on dataset detail](../images/platform_en_assets_images_qc_dataset_summary_a9595bb537e9687ebb37c9e5e1ee8298_webp.webp)
- ![QC rule list and scope filters](../images/platform_en_assets_images_qc_rules_list_c2815dacbf4259df2800efbe9fb3bbed_webp.webp)
