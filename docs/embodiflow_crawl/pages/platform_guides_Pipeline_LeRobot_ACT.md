# ACT æ¨¡åè®­ç»æå

Source: https://io-ai.tech/platform/guides/Pipeline/LeRobot/ACT/#åèèµæ

- æ°æ®æµç¨

- LeRobot æ°æ®é

- ACT æ¨¡åè®­ç»æå

# ACT æ¨¡åè®­ç»æå

æ¬æä»ç»å¦ä½ä½¿ç¨è¾æ¬§æºè½åå¸ç Docker éåioaitech/train_act:cudaè®­ç» ACT æ¨¡åãæä¸­çè¾å¥è¾åºç®å½ãåæ°ååé»è®¤å¼ä¸éååçº¦å®ä¸è´ã

ioaitech/train_act:cuda

ææ¡£é»è®¤ä»¥ Docker Hub ä¸çioaitech/train_act:cudaä¸ºä¾ãè¥ä½ å¨ä¸­å½å¤§éè®¿é® Docker Hub è¾æ¢ï¼å¯ä½¿ç¨åä¸ºäºå®¹å¨éåæå¡åæ­¥å°åï¼å°éåååç¼æ¿æ¢ä¸ºswr.cn-east-3.myhuaweicloud.com/ioaitech/ï¼ä¾å¦ï¼swr.cn-east-3.myhuaweicloud.com/ioaitech/train_act:cudaãdocker runçå¶ä½åæ°ä¸åã

ioaitech/train_act:cuda

swr.cn-east-3.myhuaweicloud.com/ioaitech/

swr.cn-east-3.myhuaweicloud.com/ioaitech/train_act:cuda

docker run

## éç¨èå´â

ACT éåä»»å¡è¾¹çæ¸æ°ãå¨ä½æ¨¡å¼ç¸å¯¹ç¨³å®çæ¨¡ä»¿å­¦ä¹ åºæ¯ãè¥ä½ çç®æ æ¯åæåä»»å¡è®­ç»é¾è·¯ç¨³å®è·éï¼åéæ­¥è°åï¼ACT ä»ç¶æ¯å¾å¡å®çéæ©ã

æ¬æåé»è®¤ä½ å·²ç»å¨è¾æ¬§æ°æ®å¹³å°å®ææ°æ®æ æ³¨ï¼å¹¶å¯¼åºäº LeRobot æ ¼å¼æ°æ®éã

## ä¸é®å¼å§è®­ç»â

### åç½®æ¡ä»¶â

- Linux ä¸»æº

- NVIDIA é©±å¨å®è£æ­£å¸¸

- Docker å¯ç¨

- docker run --gpus allå¯æ­£å¸¸è®¿é® GPU

docker run --gpus all

å»ºè®®ååä¸æ¬¡å¿«éèªæ£ï¼

docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi

docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi

### æå°å¯è¿è¡å½ä»¤â

å°æ¬å° LeRobot æ°æ®éæè½½å°/data/inputï¼å°è®­ç»è¾åºæè½½å°/data/outputï¼

/data/input

/data/output

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name act_demo \--task_name demo_task \--num_epochs 1 \--batch_size 8

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name act_demo \--task_name demo_task \--num_epochs 1 \--batch_size 8

è¿æ¡å½ä»¤éååéªè¯æ°æ®æè½½ãæ ¼å¼è¯å«åè®­ç»æµç¨æ¯å¦æéãç¡®è®¤é¾è·¯æ²¡æé®é¢åï¼åæè®­ç»è½®æ°ååæ°è°å°æ­£å¼å¼ã

### æ¨èçä¸é®è®­ç»èæ¬â

ä¸é¢è¿ç»åæ°æ´æ¥è¿å®éè®­ç»åºæ¯ï¼éåä½ä¸ºå¬å¼ææ¡£ä¸­çé¦éæ¨¡æ¿ï¼

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name lemon_act_v1 \--task_name pick_lemon \--num_epochs 12000 \--batch_size 64 \--learning_rate 5e-5 \--chunk_size 100 \--kl_weight 10 \--hidden_dim 512 \--dim_feedforward 3200 \--batch_mode fixed_global \--gpus all

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name lemon_act_v1 \--task_name pick_lemon \--num_epochs 12000 \--batch_size 64 \--learning_rate 5e-5 \--chunk_size 100 \--kl_weight 10 \--hidden_dim 512 \--dim_feedforward 3200 \--batch_mode fixed_global \--gpus all

å¦æåªæ³ä½¿ç¨æå® GPUï¼å¯ä»¥ææåä¸è¡æ¹æ--gpus 0ã--gpus 0,1è¿ç±»å½¢å¼ã

--gpus 0

--gpus 0,1

## æ°æ®è¦æ±â

å®¹å¨å¯å¨åä¼æ£æ¥/data/input/meta/info.jsonæ¯å¦å­å¨ï¼ç¼ºå°è¯¥æä»¶æ¶ä¼ç´æ¥æ¥éå¹¶éåºãå æ­¤å¨å¯å¨è®­ç»åï¼è¯·ç¡®è®¤æ°æ®éæ ¹ç®å½è³å°åå«ä»¥ä¸ç»æï¼

/data/input/meta/info.json

your_dataset/âââ meta/â   âââ info.jsonâââ data/âââ videos/

your_dataset/âââ meta/â   âââ info.jsonâââ data/âââ videos/

å½åè®­ç»å¥å£æ¯æ LeRobotv2åv3æ°æ®ãèæ¬ä¼èªå¨è¯å«çæ¬ï¼å¹¶å¨éè¦æ¶å®æå¼å®¹å¤çä¸ä¸­é´è½¬æ¢ã

v2

v3

### ç¸æºå­æ®µâ

å¦ææ°æ®éçå¾åå­æ®µå½åç¬¦åå¸¸è§çº¦å®ï¼èæ¬ä¼èªå¨ä»meta/info.jsonä¸­æ¨æ­ç¸æºé®ãè¥ä½ çå­æ®µåè¾ç¹æ®ï¼å»ºè®®æ¾å¼ä¼ å¥ï¼

meta/info.json

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name multi_cam_exp \--task_name tron2_task \--camera_keys observation.images.cam_high,observation.images.cam_right_wrist,observation.images.cam_left_wrist \--camera_names cam_high,cam_right_wrist,cam_left_wrist

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_act:cuda \--run_name multi_cam_exp \--task_name tron2_task \--camera_keys observation.images.cam_high,observation.images.cam_right_wrist,observation.images.cam_left_wrist \--camera_names cam_high,cam_right_wrist,cam_left_wrist

å¶ä¸­ï¼

- --camera_keysæ¯ LeRobot æ°æ®ä¸­çå¾åç¹å¾é®

--camera_keys

- --camera_namesæ¯ ACT è®­ç»ä¾§ä½¿ç¨çç¸æºå

--camera_names

ä¸¤èçé¡ºåºåºä¸¥æ ¼å¯¹åºã

## å¸¸ç¨åæ°â

ä¸è¡¨ä¸éååtrain_lerobot_to_act.pyçåæ°å®ä¹ä¸è´ã

train_lerobot_to_act.py

### è®­ç»æ ¸å¿åæ°â

åæ°

é»è®¤å¼

è¯´æ

--batch_size

--batch_size

64

64

è®­ç» batch size

--num_epochs

--num_epochs

12000

12000

ä¸»è®­ç»è½®æ°

--steps

--steps

0

0

num_epochsçå¼å®¹å«åï¼ä»å¨num_epochs=0æ¶ä½¿ç¨

num_epochs

num_epochs=0

--learning_rate

--learning_rate

5e-5

5e-5

ä¸»å­¦ä¹ ç

--save_interval

--save_interval

6000

6000

ä¸­é´å­æ¡£é´é

--gpus

--gpus

all

all

ä½¿ç¨å¨é¨ GPUï¼æä¼0,1è¿ç±»åè¡¨

0,1

--batch_mode

--batch_mode

fixed_global

fixed_global

å¤å¡æ¶ä¿æå¨å± batch è¯­ä¹æ´æ¥è¿åå¡åèå¼

--num_workers

--num_workers

0

0

å®¹å¨ç¯å¢ä¸æ¨èä¿æé»è®¤ï¼éä½/dev/shmé£é©

/dev/shm

### ACT æ¨¡ååæ°â

åæ°

é»è®¤å¼

è¯´æ

--task_name

--task_name

auto

auto

èªå¨ä»æ°æ®éä¸­æ¨æ­ä»»å¡åï¼å¤±è´¥æ¶ä¼åé

--run_name

--run_name

èªå¨çæ

checkpoint å­ç®å½å

--policy_class

--policy_class

ACT

ACT

ä¸è¬ä¿æé»è®¤

--kl_weight

--kl_weight

10

10

KL é¡¹æé

--chunk_size

--chunk_size

100

100

å¨ä½ chunk é¿åº¦

--hidden_dim

--hidden_dim

512

512

Transformer éå±ç»´åº¦

--dim_feedforward

--dim_feedforward

3200

3200

åé¦å±ç»´åº¦

--seed

--seed

42

42

éæºç§å­

### æ°æ®æ¡¥æ¥åæ°â

åæ°

é»è®¤å¼

è¯´æ

--camera_keys

--camera_keys

èªå¨æ¨æ­

æå® LeRobot å¾åå­æ®µ

--camera_names

--camera_names

èªå¨çæ

æå® ACT ç¸æºå

--episode_len

--episode_len

0

0

å¼ºå¶è¦ç episode é¿åº¦

--idle_threshold

--idle_threshold

1e-4

1e-4

éæ­¢å¸§è¿æ»¤éå¼

--max_episodes

--max_episodes

0

0

ä»è½¬æ¢å N ä¸ª episodeï¼éå smoke test

--convert_workers

--convert_workers

0

0

è½¬æ¢é¶æ®µå¹¶å worker æ°

--keep_converted_hdf5

--keep_converted_hdf5

å³é­

ä¿çä¸­é´ HDF5 æä»¶ï¼ä¾¿äºææ¥é®é¢

## è¾åºç»æâ

è®­ç»è¾åºä¼åå¥æè½½ç/data/outputãå¸åäº§ç©å¦ä¸ï¼

/data/output

/path/to/output/âââ checkpoints/â   âââ <run_name>/â       âââ policy_best.ckptâ       âââ policy_last.ckptâ       âââ dataset_stats.pklâââ manifest.json

/path/to/output/âââ checkpoints/â   âââ <run_name>/â       âââ policy_best.ckptâ       âââ policy_last.ckptâ       âââ dataset_stats.pklâââ manifest.json

å¶ä¸­ï¼

- policy_best.ckptæ¯è®­ç»è¿ç¨ä¸­è¡¨ç°è¾å¥½ç checkpoint

policy_best.ckpt

- policy_last.ckptæ¯æåä¸æ¬¡ä¿å­ç checkpoint

policy_last.ckpt

- dataset_stats.pklæ¯è®­ç»æ¶ä½¿ç¨çæ°æ®ç»è®¡ä¿¡æ¯

dataset_stats.pkl

- manifest.jsonè®°å½æ¬æ¬¡è®­ç»çå³é®ä¿¡æ¯

manifest.json

## å¤å¡è®­ç»å»ºè®®â

å½åå®ç°å·²ç»æå¤å¡è®­ç»è·¯å¾å°è£å¨å®¹å¨åé¨ï¼å¸¸è§æåµä¸ä¸éè¦æå¨æ¹torchrunå½ä»¤ãå»ºè®®éµå¾ªä»¥ä¸ååï¼

torchrun

- ä¼åä½¿ç¨--batch_mode fixed_globalï¼æ´å®¹æä¸åå¡ç»æå¯¹é½

--batch_mode fixed_global

- å®¹å¨å--num_workerså»ºè®®ä¿æ0

--num_workers

0

- åæ¬¡å¤å¡å®éªå¯ä»¥åéå--max_episodes 10åå¿«ééªè¯

--max_episodes 10

è¥ä½ æç¡®ç¥éèªå·±è¦è¿½æ±æ´é«ååï¼åèèåæ¢--batch_mode fixed_per_gpuã

--batch_mode fixed_per_gpu

## å¸¸è§é®é¢â

### 1. å®¹å¨å¯å¨åæç¤ºæ¾ä¸å°æ°æ®éâ

åæ£æ¥ä¸¤ä»¶äºï¼

- å®¿ä¸»æºè·¯å¾æ¯å¦æ­£ç¡®æè½½å°/data/input

/data/input

- /data/input/meta/info.jsonæ¯å¦å­å¨

/data/input/meta/info.json

ç¼ºå°info.jsonæ¶ï¼å®¹å¨ä¼ç´æ¥éåºï¼è¿éå¸¸ä¸æ¯è®­ç»ä»£ç é®é¢ï¼èæ¯æ°æ®è·¯å¾æç®å½å±çº§ä¸æ­£ç¡®ã

info.json

### 2. å¤å¡è®­ç»åºç° DataLoader å¼å¸¸æ NCCL è¶æ¶â

ä¼åå°è¯ä»¥ä¸åæ³ï¼

- ä¿æ--num_workers 0

--num_workers 0

- å°--convert_workersè°å°å°2æ4

--convert_workers

2

4

- åç¼©å°æ°æ®è§æ¨¡ï¼ç¨--max_episodesåç­æµç¨éªè¯

--max_episodes

### 3. å¦ä½è®¾ç½®task_nameâ

task_name

å¦ææ°æ®éä¸­çä»»å¡å­æ®µå®æ´ï¼--task_name autoéå¸¸å³å¯å·¥ä½ãè¥ä½ çæ°æ®éä»»å¡å®ä¹è¾å¤æï¼å»ºè®®æ¾å¼æå®ä»»å¡åï¼ä¾¿äºåç»­ç®¡çè¾åºç®å½åå®éªè®°å½ã

--task_name auto

### 4. é¦æ¬¡è®­ç»éåº¦åæ¢â

éåæå»ºé¶æ®µå·²ç»é¢ä¸è½½äºResNet18æéï¼ä½æ°æ®è½¬æ¢åé¦è½®æ°æ®å è½½ä»ç¶éè¦æ¶é´ãåªè¦æ¥å¿å¨æç»­æ¨è¿ï¼éå¸¸å±äºæ­£å¸¸ç°è±¡ã

ResNet18

## å®è·µå»ºè®®â

- åç¨1ä¸ª epoch æå°é episode åæµç¨éªè¯ï¼åè·æ­£å¼è®­ç»

1

- åºå®ä¸ç»åºç¡åæ°ä½ä¸ºå¯¹ç§ç»ï¼åç»­æ¯æ¬¡åªè°æ´å°éåé

- ä¸è¦åªçæåä¸ä¸ª checkpointï¼å»ºè®®åæ¶æ¯è¾è¥å¹²ä¸­é´ checkpoint çå®éææ

## åèèµæâ

- ACT å®æ¹è®ºæ

- LeRobot å®æ¹ææ¡£