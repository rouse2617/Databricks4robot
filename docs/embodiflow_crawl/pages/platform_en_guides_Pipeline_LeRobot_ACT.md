# ACT model training guide

Source: https://io-ai.tech/platform/en/guides/Pipeline/LeRobot/ACT/#references

- Data Pipeline

- LeRobot Dataset

- ACT model training guide

# ACT model training guide

This guide explains how to train ACT with the Docker imageioaitech/train_act:cudapublished byIO-AI.TECH. Mount paths, argument names, and defaults match the training scripts inside the image.

ioaitech/train_act:cuda

Images are published onDocker Hubunder theioaitechorganization (e.g.ioaitech/train_act:cuda).

ioaitech

ioaitech/train_act:cuda

## When to use ACTâ

ACT fits imitation-learning setups where the task boundary is clear and the action pattern is relatively stable. If your first goal is to get a single-task training loop working reliably and then tune hyperparameters, ACT remains a practical choice.

This guide assumes you have already labeled data on the EmbodiFlow platform and exported it in LeRobot format.

## One-command trainingâ

### Prerequisitesâ

- Linux host

- Working NVIDIA driver

- Docker installed

- docker run --gpus allcan see the GPU

docker run --gpus all

Quick check:

docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi

docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi

### Minimal runâ

Mount your local LeRobot dataset at/data/inputand outputs at/data/output:

/data/input

/data/output

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name act_demo \--task_name demo_task \--num_epochs 1 \--batch_size 8

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name act_demo \--task_name demo_task \--num_epochs 1 \--batch_size 8

Use this to verify mounts, format detection, and the training path end to end. After that, increase epochs and adjust hyperparameters for a real run.

### Recommended one-shot scriptâ

This template is closer to a typical production-style run:

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name lemon_act_v1 \--task_name pick_lemon \--num_epochs 12000 \--batch_size 64 \--learning_rate 5e-5 \--chunk_size 100 \--kl_weight 10 \--hidden_dim 512 \--dim_feedforward 3200 \--batch_mode fixed_global \--gpus all

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name lemon_act_v1 \--task_name pick_lemon \--num_epochs 12000 \--batch_size 64 \--learning_rate 5e-5 \--chunk_size 100 \--kl_weight 10 \--hidden_dim 512 \--dim_feedforward 3200 \--batch_mode fixed_global \--gpus all

To pin specific GPUs, change the last line to e.g.--gpus 0or--gpus 0,1.

--gpus 0

--gpus 0,1

## Data requirementsâ

The container checks for/data/input/meta/info.jsonat startup; if it is missing, the run exits immediately. Your dataset root should look like:

/data/input/meta/info.json

your_dataset/âââ meta/â   âââ info.jsonâââ data/âââ videos/

your_dataset/âââ meta/â   âââ info.jsonâââ data/âââ videos/

The training entrypoint supports LeRobotv2andv3datasets. It detects the version and applies compatibility handling and intermediate conversion when needed.

### Camera fieldsâ

If image feature names follow common conventions, camera keys are inferred frommeta/info.json. For unusual naming, pass them explicitly:

meta/info.json

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name multi_cam_exp \--task_name tron2_task \--camera_keys observation.images.cam_high,observation.images.cam_right_wrist,observation.images.cam_left_wrist \--camera_names cam_high,cam_right_wrist,cam_left_wrist

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name multi_cam_exp \--task_name tron2_task \--camera_keys observation.images.cam_high,observation.images.cam_right_wrist,observation.images.cam_left_wrist \--camera_names cam_high,cam_right_wrist,cam_left_wrist

- --camera_keys: LeRobot image feature keys

--camera_keys

- --camera_names: ACT-side camera names

--camera_names

The two lists must stay aligned in order.

## Common argumentsâ

The following tables match the argument definitions intrain_lerobot_to_act.pyinside the image.

train_lerobot_to_act.py

### Core trainingâ

Argument

Default

Description

--batch_size

--batch_size

64

64

Training batch size

--num_epochs

--num_epochs

12000

12000

Number of training epochs

--steps

--steps

0

0

Alias fornum_epochs; used only whennum_epochs=0

num_epochs

num_epochs=0

--learning_rate

--learning_rate

5e-5

5e-5

Main learning rate

--save_interval

--save_interval

6000

6000

Checkpoint save interval (epochs)

--gpus

--gpus

all

all

All GPUs, or a list like0,1

0,1

--batch_mode

--batch_mode

fixed_global

fixed_global

Multi-GPU global batch semantics closer to single-GPU reference

--num_workers

--num_workers

0

0

Recommended in containers to reduce/dev/shmpressure

/dev/shm

### ACT modelâ

Argument

Default

Description

--task_name

--task_name

auto

auto

Infer primary task from dataset; falls back on failure

--run_name

--run_name

auto

Checkpoint subdirectory name

--policy_class

--policy_class

ACT

ACT

Usually leave default

--kl_weight

--kl_weight

10

10

KL loss weight

--chunk_size

--chunk_size

100

100

Action chunk length

--hidden_dim

--hidden_dim

512

512

Transformer hidden size

--dim_feedforward

--dim_feedforward

3200

3200

FFN hidden size

--seed

--seed

42

42

Random seed

### Data bridgeâ

Argument

Default

Description

--camera_keys

--camera_keys

inferred

LeRobot image keys

--camera_names

--camera_names

derived

ACT camera names

--episode_len

--episode_len

0

0

Force episode length;0= auto

0

--idle_threshold

--idle_threshold

1e-4

1e-4

Idle-frame filter threshold

--max_episodes

--max_episodes

0

0

Convert only first N episodes (smoke test)

--convert_workers

--convert_workers

0

0

Parallel workers for conversion

--keep_converted_hdf5

--keep_converted_hdf5

off

Keep intermediate HDF5 for debugging

## Outputsâ

Artifacts are written under the mounted/data/output:

/data/output

/path/to/output/âââ checkpoints/â   âââ <run_name>/â       âââ policy_best.ckptâ       âââ policy_last.ckptâ       âââ dataset_stats.pklâââ manifest.json

/path/to/output/âââ checkpoints/â   âââ <run_name>/â       âââ policy_best.ckptâ       âââ policy_last.ckptâ       âââ dataset_stats.pklâââ manifest.json

- policy_best.ckpt: best checkpoint during training

policy_best.ckpt

- policy_last.ckpt: last saved checkpoint

policy_last.ckpt

- dataset_stats.pkl: statistics used for training

dataset_stats.pkl

- manifest.json: run metadata

manifest.json

## Multi-GPU notesâ

Multi-GPU launch is handled inside the container; you normally do not need to hand-rolltorchrun. Suggested practice:

torchrun

- Prefer--batch_mode fixed_globalfor easier comparison with single-GPU runs

--batch_mode fixed_global

- Keep--num_workers 0in containers

--num_workers 0

- For a first multi-GPU test, add--max_episodes 10

--max_episodes 10

Switch to--batch_mode fixed_per_gpuonly when you intentionally prioritize throughput.

--batch_mode fixed_per_gpu

## FAQâ

### 1. Container says dataset not foundâ

Check:

- Host path is mounted to/data/input

/data/input

- /data/input/meta/info.jsonexists

/data/input/meta/info.json

Missinginfo.jsonusually means a wrong path or directory layout, not a bug in the trainer.

info.json

### 2. DataLoader errors or NCCL timeouts on multi-GPUâ

Try:

- Keep--num_workers 0

--num_workers 0

- Lower--convert_workersto2or4

--convert_workers

2

4

- Shorten the pipeline with--max_episodesfor a dry run

--max_episodes

### 3. Choosingtask_nameâ

task_name

If task metadata is complete,--task_name autois usually enough. For complex task definitions, set an explicit name to keep outputs and logs organized.

--task_name auto

### 4. First run feels slowâ

ResNet18weights are baked into the image, but conversion and the first data pass still take time. Steady log progress is expected.

ResNet18

## Practical tipsâ

- Validate the pipeline with 1 epoch or a small--max_episodesbefore a long run

--max_episodes

- Keep a fixed baseline config and change one knob at a time

- Compare several checkpoints, not only the last one

## Referencesâ

- ACT paper

- LeRobot documentation