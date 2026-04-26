# Pi0 and Pi0.5 model fine-tuning guide

Source: https://io-ai.tech/platform/en/guides/Pipeline/LeRobot/Pi0/#references

- Data Pipeline

- LeRobot Dataset

- Pi0 and Pi0.5 model fine-tuning guide

# Pi0 and Pi0.5 model fine-tuning guide

This guide explains how tofine-tunePi0andPi0.5with Docker images published byIO-AI.TECH, on top of base checkpoints such aspi0_base/pi05_base. Commands, mount conventions, and argument names match the OpenPI wrappertrain_lerobot.pyinside the image.

train_lerobot.py

If you want the official OpenPIfine-tuningworkflow without hand-building the environment, this is the path that matches day-to-day deployment practice.

## Why this pathâ

Pi0 / Pi0.5 arefine-tunedwith theOpenPIstack (JAX under the hood) from public base weights. Once you have a LeRobot dataset, the most direct approach is to use:

- ioaitech/train_openpi:pi0

ioaitech/train_openpi:pi0

- ioaitech/train_openpi:pi05

ioaitech/train_openpi:pi05

Both images ship the dependencies needed forfine-tuningand use:

- Dataset mounted at/data/input

/data/input

- Outputs (checkpoints, etc.) at/data/output

/data/output

Images are published onDocker Hub:ioaitech/train_openpi:pi0andioaitech/train_openpi:pi05.

ioaitech/train_openpi:pi0

ioaitech/train_openpi:pi05

## One-command fine-tuningâ

### Prerequisitesâ

- Linux host

- Working NVIDIA driver

- Docker

- docker run --gpus allworks

docker run --gpus all

GPU sanity check:

docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi

docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi

### Pi0â

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 8 \--steps 20000 \--save_interval 1000 \--learning_rate 2.5e-5 \--action_horizon 50 \--prompt "pick up the object"

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 8 \--steps 20000 \--save_interval 1000 \--learning_rate 2.5e-5 \--action_horizon 50 \--prompt "pick up the object"

### Pi0.5â

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi05 \--batch_size 8 \--steps 20000 \--save_interval 1000 \--learning_rate 2.5e-5 \--action_horizon 50 \--prompt "pick up the object"

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi05 \--batch_size 8 \--steps 20000 \--save_interval 1000 \--learning_rate 2.5e-5 \--action_horizon 50 \--prompt "pick up the object"

The main difference is the image tag. The container selects base weights andPi0Configaccording to theMODEL_TYPEbaked into the image at build time.

Pi0Config

MODEL_TYPE

### Minimal smoke runâ

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 1 \--steps 1000

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 1 \--steps 1000

After this succeeds, increasebatch_size,steps, and otherfine-tuninghyperparameters.

batch_size

steps

## Data requirementsâ

The entrypoint requires/data/input/meta/info.json. Your dataset root should include:

/data/input/meta/info.json

your_dataset/âââ meta/â   âââ info.jsonâââ data/âââ videos/

your_dataset/âââ meta/â   âââ info.jsonâââ data/âââ videos/

Thefine-tuningwrapper automatically:

- Detects the LeRobot dataset version

- Convertsv3layouts to av2.1-compatible tree when needed for the pipeline

- Ensuresepisodes_stats.jsonlforv2.1when missing

episodes_stats.jsonl

- Computes normalization statistics

- Symlinks the mounted tree into the LeRobot cache for OpenPI

## Pi0 vs Pi0.5â

They share onefine-tuningwrapper but differ in model config and weights:

- Pi0 loadspi0_basecheckpoints

pi0_base

- Pi0.5 loadspi05_basecheckpoints

pi05_base

- Pi0.5 usesPi0Config(pi05=True)(state/token layout differs from Pi0)

Pi0Config(pi05=True)

- Pi0.5 uses a larger defaultmax_token_lenfor richer conditioning

max_token_len

Start withPi0to validate the pipeline; switch toioaitech/train_openpi:pi05when you intentionally want the Pi0.5 base and behavior.

ioaitech/train_openpi:pi05

## Common argumentsâ

The arguments below matchtrain_lerobot.pyin the image:

train_lerobot.py

Argument

Default

Description

--batch_size

--batch_size

1

1

Batch per optimization step; raised if needed to divide JAX device count

--steps

--steps

1000

1000

Number of optimization steps

--gpus

--gpus

all

all

All GPUs or e.g.0,1

0,1

--prompt

--prompt

empty

Default language prompt when the dataset has no task text

--save_interval

--save_interval

500

500

Checkpoint interval

--learning_rate

--learning_rate

empty

Omit to use peak LR2.5e-5

2.5e-5

--fsdp_devices

--fsdp_devices

auto

auto

FSDP device count; auto from GPU count

--lora

--lora

auto

auto

LoRA on by default for single GPU, off for multi-GPU

--ema_decay

--ema_decay

empty

EMA; disabled under LoRA by default to save VRAM

--action_horizon

--action_horizon

50

50

Action sequence length

## Fine-tuning behaviorâ

The wrapper picks strategies automatically:

- Single GPU:--lora autotends to enable LoRA to save memory

--lora auto

- Multi GPU:--fsdp_devices autoenables FSDP-style sharding

--fsdp_devices auto

- Ifbatch_sizeis smaller than the device count or not divisible, it is bumped to a valid value

batch_size

So the one-liners above emphasize âit runsâ before you tune every JAX detail.

### About--promptâ

--prompt

If episodes include task strings, those take precedence duringfine-tuning.--promptis only a fallback when the dataset has no task field. Treat it as optional unless you know your export lacks language metadata.

--prompt

## Outputsâ

Checkpoints go under the mounted/data/output. The embeddedTrainConfigfixes:

/data/output

TrainConfig

- name = docker_train

name = docker_train

- exp_name = train

exp_name = train

So the default checkpoint directory is:

/path/to/output/docker_train/train/

/path/to/output/docker_train/train/

Normalization stats are written under the asset directories configured duringfine-tuningfor reuse in later runs or inference.

## Suggested workflowsâ

### 1. Short validation on a new datasetâ

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 1 \--steps 1000 \--save_interval 200

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 1 \--steps 1000 \--save_interval 200

Confirm logs and that checkpoints appear before scaling upfine-tuning.

### 2. Conservative single-GPU runâ

Keep the default auto policy; do not force LoRA off at first:

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 4 \--steps 20000 \--save_interval 1000 \--action_horizon 50

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 4 \--steps 20000 \--save_interval 1000 \--action_horizon 50

### 3. Multi-GPUâ

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi05 \--gpus 0,1,2,3 \--batch_size 16 \--steps 30000 \--fsdp_devices 4 \--save_interval 1000

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi05 \--gpus 0,1,2,3 \--batch_size 16 \--steps 30000 \--fsdp_devices 4 \--save_interval 1000

## FAQâ

### 1. âNo LeRobot datasetâ at startupâ

Verify the host mount to/data/inputand the presence of/data/input/meta/info.json.

/data/input

/data/input/meta/info.json

### 2. Different behavior on one vs many GPUsâ

By design: LoRA, FSDP, and batch-size adjustment depend on device count.

### 3. Why is the output folder fixed?â

The wrapper currently hard-codesdocker_train/trainunder/data/output. Finer experiment naming may be added elsewhere; the docs reflect the image behavior as shipped.

docker_train/train

/data/output

### 4. Do I runcompute_norm_statsmanually?â

compute_norm_stats

No. The wrapper computes and saves normalization statistics beforefine-tuningstarts.

### 5. Pi0 or Pi0.5?â

Use Pi0 to stabilize the pipeline first. Use Pi0.5 when you explicitly want that base and configuration.

## Practical tipsâ

- Validate dataset layout before longfine-tuningjobs

- Short runs before long runs

- On single GPU, accept defaults before overriding LoRA/FSDP

- Keep separate output roots when comparing Pi0 vs Pi0.5

## Referencesâ

- OpenPI on GitHub

- Pi0 paper

- LeRobot dataset spec