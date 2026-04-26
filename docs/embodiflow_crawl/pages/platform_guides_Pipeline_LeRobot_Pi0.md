# Pi0 ä¸ Pi0.5 æ¨¡åå¾®è°æå

Source: https://io-ai.tech/platform/guides/Pipeline/LeRobot/Pi0/#åèèµæ

- æ°æ®æµç¨

- LeRobot æ°æ®é

- Pi0 ä¸ Pi0.5 æ¨¡åå¾®è°æå

# Pi0 ä¸ Pi0.5 æ¨¡åå¾®è°æå

æ¬æä»ç»å¦ä½ä½¿ç¨è¾æ¬§æºè½åå¸ç Docker éåï¼å¨pi0_base/pi05_baseç­é¢è®­ç»æéåºç¡ä¸å¾®è°Pi0 ä¸ Pi0.5 æ¨¡åãæä¸­çå½ä»¤ãç®å½çº¦å®ååæ°åä¸éåå OpenPI å°è£å¥å£train_lerobot.pyä¸è´ã

train_lerobot.py

å¦æä½ çç®æ æ¯å¤ç°å®æ¹ OpenPIå¾®è°é¾è·¯ï¼åæ¶åå¸æç´æ¥ä½¿ç¨å·²ç»åå¤å¥½çéåç¯å¢ï¼é£ä¹è¿æ¡è·¯å¾æçäºï¼ä¹ææ¥è¿å®éå¯å¤ç¨çé¨ç½²æ¹å¼ã

## ä¸ºä»ä¹æ¨èè¿æ¡è·¯å¾â

Pi0 / Pi0.5 åºäº OpenPI å®æ¹æ ä¸å¬å¼åºåº§æ£æ¥ç¹è¿è¡å¾®è°ï¼åºå±ä¸º JAX å®ç°ï¼ãå¯¹äºå·²ç»åå¤å¥½ LeRobot æ°æ®éçä½¿ç¨èæ¥è¯´ï¼æç´æ¥çåæ³ä¸æ¯åæå·¥æ­å»ºå®æ´ç¯å¢ï¼èæ¯ç´æ¥ä½¿ç¨ï¼

- ioaitech/train_openpi:pi0

ioaitech/train_openpi:pi0

- ioaitech/train_openpi:pi05

ioaitech/train_openpi:pi05

è¿ä¸¤ä¸ªéåå·²ç»åå«äºå¾®è°æéä¾èµï¼å¹¶çº¦å®ï¼

- è¾å¥æ°æ®æè½½å°/data/input

/data/input

- è¾åºï¼æ£æ¥ç¹ç­ï¼åå¥/data/output

/data/output

ææ¡£é»è®¤ä»¥ Docker Hub ä¸çioaitech/train_openpi:pi0/ioaitech/train_openpi:pi05ä¸ºä¾ãè¥ä½ å¨ä¸­å½å¤§éè®¿é® Docker Hub è¾æ¢ï¼å¯å°éååç¼æ¿æ¢ä¸ºswr.cn-east-3.myhuaweicloud.com/ioaitech/ï¼ä¾å¦ï¼

ioaitech/train_openpi:pi0

ioaitech/train_openpi:pi05

swr.cn-east-3.myhuaweicloud.com/ioaitech/

- swr.cn-east-3.myhuaweicloud.com/ioaitech/train_openpi:pi0

swr.cn-east-3.myhuaweicloud.com/ioaitech/train_openpi:pi0

- swr.cn-east-3.myhuaweicloud.com/ioaitech/train_openpi:pi05docker runçå¶ä½åæ°ä¸åã

swr.cn-east-3.myhuaweicloud.com/ioaitech/train_openpi:pi05

docker run

## ä¸é®å¼å§å¾®è°â

### åç½®æ¡ä»¶â

- Linux ä¸»æº

- NVIDIA é©±å¨å®è£æ­£å¸¸

- Docker å¯ç¨

- docker run --gpus allå¯æ­£å¸¸è®¿é® GPU

docker run --gpus all

å»ºè®®ååä¸æ¬¡ GPU èªæ£ï¼

docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi

docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi

### Pi0 ä¸é®å¾®è°â

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 8 \--steps 20000 \--save_interval 1000 \--learning_rate 2.5e-5 \--action_horizon 50 \--prompt "pick up the object"

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 8 \--steps 20000 \--save_interval 1000 \--learning_rate 2.5e-5 \--action_horizon 50 \--prompt "pick up the object"

### Pi0.5 ä¸é®å¾®è°â

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi05 \--batch_size 8 \--steps 20000 \--save_interval 1000 \--learning_rate 2.5e-5 \--action_horizon 50 \--prompt "pick up the object"

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi05 \--batch_size 8 \--steps 20000 \--save_interval 1000 \--learning_rate 2.5e-5 \--action_horizon 50 \--prompt "pick up the object"

ä¸é¢ä¸¤æ¡å½ä»¤çä¸»è¦åºå«å¨äºéåæ ç­¾ï¼å¯¹åºä¸åçåºåº§æéä¸æ¨¡åéç½®ã

### æå°å¯è¿è¡å½ä»¤â

å¦æä½ åªæ¯æ³åéªè¯æ°æ®æè½½åå¾®è°é¾è·¯æ¯å¦æéï¼å¯ä»¥æå½ä»¤ç¼©åä¸ºï¼

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 1 \--steps 1000

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 1 \--steps 1000

é¦æ¬¡è·éä¹åï¼åéæ­¥å¢åbatch_sizeãstepsåå¶å®å¾®è°è¶åæ°ã

batch_size

steps

## æ°æ®è¦æ±â

å®¹å¨å¥å£ä¼æ£æ¥/data/input/meta/info.jsonæ¯å¦å­å¨ï¼ç¼ºå°è¯¥æä»¶æ¶ä¼ç´æ¥éåºãå æ­¤è¯·ç¡®ä¿ä½ ç LeRobot æ°æ®éæ ¹ç®å½è³å°åå«ï¼

/data/input/meta/info.json

your_dataset/âââ meta/â   âââ info.jsonâââ data/âââ videos/

your_dataset/âââ meta/â   âââ info.jsonâââ data/âââ videos/

å½åå¾®è°å°è£ä¼èªå¨å¤çä»¥ä¸äºé¡¹ï¼

- è¯å« LeRobot æ°æ®éçæ¬

- å¿è¦æ¶å°v3æ°æ®è½¬æ¢ä¸ºå¼å®¹å¾®è°æµç¨çv2.1å¸å±

v3

v2.1

- ä¸ºv2.1æ°æ®è¡¥é½episodes_stats.jsonl

v2.1

episodes_stats.jsonl

- çæå½ä¸åç»è®¡ä¿¡æ¯

- å°æè½½çæ°æ®éé¾æ¥å° LeRobot cacheï¼ä¾ OpenPI è¯»å

## Pi0 ä¸ Pi0.5 çåºå«â

è¿ä¸¤ç§æ¨¡åå±ç¨ä¸å¥å¾®è°å°è£ï¼ä½åºå±éç½®å¹¶ä¸ç¸åã

å¨å½åä»£ç ä¸­ï¼å·®å¼ä¸»è¦ä½ç°å¨ä»¥ä¸å ç¹ï¼

- Pi0 ä½¿ç¨pi0_baseé¢è®­ç»æé

pi0_base

- Pi0.5 ä½¿ç¨pi05_baseé¢è®­ç»æé

pi05_base

- Pi0.5 ä¼å¯ç¨Pi0Config(pi05=True)ï¼å¶ç¶æè¾å¥å token éç½®ä¸ Pi0 ä¸å

Pi0Config(pi05=True)

- Pi0.5 é»è®¤max_token_lenæ´é¿ï¼æ´éåéè¦æ´å¼ºè¡¨è¾¾è½åçåºæ¯

max_token_len

è¥ä½ å¸æåå¿«éå¤ç°å¾®è°æµç¨ï¼å»ºè®®ä» Pi0 å¼å§ï¼è¥ä½ å·²ç»ææç¡®çä»»å¡éæ±ï¼å¹¶ä¸è®¡åç´æ¥ä½¿ç¨ Pi0.5 çè½åè¾¹çï¼ååå°ioaitech/train_openpi:pi05ã

ioaitech/train_openpi:pi05

## å¸¸ç¨åæ°â

ä»¥ä¸åæ°ä¸éååtrain_lerobot.pyçå®ä¹ä¸è´ã

train_lerobot.py

åæ°

é»è®¤å¼

è¯´æ

--batch_size

--batch_size

1

1

æ¯ä¸ªä¼å step ç batchï¼æç»ä¼èªå¨è°æ´ä¸ºè½æ´é¤ JAX è®¾å¤æ°

--steps

--steps

1000

1000

ä¼åæ­¥æ°

--gpus

--gpus

all

all

ä½¿ç¨å¨é¨ GPUï¼æä¼0,1è¿ç±»åè¡¨

0,1

--prompt

--prompt

ç©º

å½æ°æ®éæ²¡æä»»å¡ææ¬æ¶ä½¿ç¨çé»è®¤è¯­è¨æç¤º

--save_interval

--save_interval

500

500

checkpoint ä¿å­é´é

--learning_rate

--learning_rate

ç©º

ä¸ä¼ æ¶ä½¿ç¨é»è®¤å³°å¼å­¦ä¹ ç2.5e-5

2.5e-5

--fsdp_devices

--fsdp_devices

auto

auto

èªå¨å³å®æ¯å¦å¯ç¨ FSDPï¼ä»¥ååçè®¾å¤æ°

--lora

--lora

auto

auto

åå¡é»è®¤å¯ç¨ LoRAï¼å¤å¡é»è®¤å³é­

--ema_decay

--ema_decay

ç©º

EMA è¡°åï¼LoRA æ¨¡å¼ä¸é»è®¤å³é­ï¼ä»¥èçæ¾å­

--action_horizon

--action_horizon

50

50

å¨ä½åºåé¿åº¦

## å¾®è°è¡ä¸ºè¯´æâ

å½åå°è£å¯¹åå¡åå¤å¡åäºèªå¨ç­ç¥éæ©ï¼

- åå¡æ¶ï¼--lora autoé»è®¤ä¼å¯ç¨ LoRAï¼ä»¥éä½æ¾å­åå

--lora auto

- å¤å¡æ¶ï¼--fsdp_devices autoä¼èªå¨åæ¢å° FSDP è·¯å¾

--fsdp_devices auto

- batch_sizeè¥å°äºè®¾å¤æ°ï¼èæ¬ä¼èªå¨ä¸è°å°å¯æ´é¤çå¼

batch_size

è¿æå³çææ¡£ä¸­çä¸é®å½ä»¤é¦åå¼ºè°âè½è·èµ·æ¥âï¼èä¸æ¯è¦æ±ç¨æ·ä¸å¼å§å°±å¤çå¨é¨ JAX ç»èã

### å³äºpromptâ

prompt

å¦ææ°æ®éä¸­æ¬èº«åå«ä»»å¡ææ¬ï¼å¾®è°æ¶ä¼ä¼åä½¿ç¨æ°æ®ä¸­çä»»å¡ä¿¡æ¯ãåªæå½æ°æ®éç¼ºå°ä»»å¡å­æ®µæ¶ï¼--promptæä¼ä½ä¸ºé»è®¤æä»¤çæã

--prompt

å æ­¤ï¼å»ºè®®æ--promptçè§£ä¸ºååºåæ°ï¼èä¸æ¯ææåºæ¯é½å¿é¡»æå·¥å¡«åçåæ°ã

--prompt

## è¾åºç»æâ

å¾®è°è¾åºä¼åå¥æè½½ç/data/outputãå½åTrainConfigåºå®ä½¿ç¨ï¼

/data/output

TrainConfig

- name = docker_train

name = docker_train

- exp_name = train

exp_name = train

å æ­¤ checkpoint é»è®¤ä¼è½å°ï¼

/path/to/output/docker_train/train/

/path/to/output/docker_train/train/

å¾®è°è¿ç¨ä¸­è¿ä¼çæå½ä¸åç»è®¡ä¿¡æ¯ï¼ä¿å­å¨èµäº§ç®å½ä¸ï¼ä¾åç»­å¾®è°ä¸æ¨çä½¿ç¨ã

## æ¨èç¨æ³â

### 1. ååå°è§æ¨¡éªè¯â

ç¬¬ä¸æ¬¡è·æ°æ°æ®éæ¶ï¼å»ºè®®åç¨ä¸é¢è¿ç±»éç½®åé¾è·¯éªè¯ï¼

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 1 \--steps 1000 \--save_interval 200

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 1 \--steps 1000 \--save_interval 200

ç¡®è®¤æ¥å¿æ­£å¸¸ãè¾åºç®å½åºç° checkpoint åï¼åæå¾®è°è§æ¨¡è°å¤§ã

### 2. åå¡ä¼åç¨³å¦¥éç½®â

å¦æä½ åªæåå¼  GPUï¼å»ºè®®åä¿çé»è®¤èªå¨ç­ç¥ï¼ä¸è¦ä¸å¼å§å°±å¼ºè¡å³é­ LoRAï¼

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 4 \--steps 20000 \--save_interval 1000 \--action_horizon 50

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi0 \--batch_size 4 \--steps 20000 \--save_interval 1000 \--action_horizon 50

### 3. å¤å¡å¾®è°â

è¥ä½ å·²ç»å·å¤å¤å¡ç¯å¢ï¼å¯ä»¥æ¾å¼æå® GPU å FSDP è®¾å¤æ°ï¼

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi05 \--gpus 0,1,2,3 \--batch_size 16 \--steps 30000 \--fsdp_devices 4 \--save_interval 1000

docker run --rm --gpus all \-v /path/to/lerobot_dataset:/data/input \-v /path/to/output:/data/output \ioaitech/train_openpi:pi05 \--gpus 0,1,2,3 \--batch_size 16 \--steps 30000 \--fsdp_devices 4 \--save_interval 1000

## å¸¸è§é®é¢â

### 1. å®¹å¨å¯å¨åæç¤ºæ¾ä¸å° LeRobot æ°æ®éâ

åæ£æ¥ï¼

- å®¿ä¸»æºè·¯å¾æ¯å¦æ­£ç¡®æè½½å°/data/input

/data/input

- /data/input/meta/info.jsonæ¯å¦å­å¨

/data/input/meta/info.json

å®¹å¨å¥å£ä¼å¨æå¼å§åè¿ä¸ªæ£æ¥ï¼è·¯å¾ä¸å¯¹æ¶ä¸ä¼è¿å¥ å¾®è°é¶æ®µã

### 2. ä¸ºä»ä¹åæ ·çå½ä»¤å¨åå¡åå¤å¡ä¸è¡ä¸ºä¸åâ

è¿æ¯å°è£èæ¬çè®¾è®¡ä½¿ç¶ãtrain_lerobot.pyä¼æ ¹æ®è®¾å¤æ°éèªå¨å³å®æ¯å¦å¯ç¨ LoRAãæ¯å¦ä½¿ç¨ FSDPï¼å¹¶å¯¹batch_sizeåå¼å®¹è°æ´ï¼ä»¥åå°åæ¬¡ä½¿ç¨çè¯éææ¬ã

train_lerobot.py

batch_size

### 3. ä¸ºä»ä¹è¾åºç®å½ä¸æ¯èªå·±å½åçâ

å½åå°è£ä¸­è¾åºç®å½å½ååºå®ä¸º/data/output/docker_train/trainãå¦æåç»­éè¦æå®éªååºåç®å½ï¼å¯ä»¥å¨å¹³å°ä¾§æåç»­çæ¬éåå¢å æ´ç»çå½åå¥å£ï¼ç®åææ¡£ä»¥éååè¡ä¸ºä¸ºåã

/data/output/docker_train/train

### 4. éè¦æå·¥åè·å½ä¸åç»è®¡åâ

ä¸éè¦ãå½åå°è£ä¼å¨å¾®è°å¼å§åèªå¨è®¡ç®å¹¶ä¿å­å½ä¸åç»è®¡ä¿¡æ¯ï¼è¿ä¹æ¯æ¨èç´æ¥ä½¿ç¨éåçåå ä¹ä¸ã

### 5. Pi0 å Pi0.5 è¯¥éåªä¸ªâ

å¦æä½ å½åçç®æ æ¯åç¨³å®å¤ç°å®æ´å¾®è°æµç¨ï¼å»ºè®®åç¨ Pi0ï¼å¦æä½ å·²ç»ç¡®è®¤è¦ä½¿ç¨ Pi0.5ï¼å¹¶ä¸å¸æç´æ¥æ²¿ç Pi0.5 çåºåº§ç»§ç»­å¾®è°ï¼å°±ç´æ¥ä½¿ç¨ioaitech/train_openpi:pi05ã

ioaitech/train_openpi:pi05

## å®è·µå»ºè®®â

- åéªè¯æ°æ®éç»æï¼åå¯å¨æ­£å¼å¾®è°

- åè·ç­æµç¨ï¼åå¢å ä¼åæ­¥æ°

- åå¡åºæ¯åæ¥åé»è®¤èªå¨ç­ç¥ï¼é¿åè¿æ©æå·¥å¹²é¢ LoRA/FSDP

- Pi0 ä¸ Pi0.5 åå«ä¿çç¬ç«è¾åºç®å½ï¼ä¾¿äºåç»­æ¯è¾ææ

## åèèµæâ

- OpenPI å®æ¹ GitHub

- Pi0 è®ºæ: Ïâ: A Flow-based Vision-Language-Action Model

- LeRobot æ°æ®éè§è