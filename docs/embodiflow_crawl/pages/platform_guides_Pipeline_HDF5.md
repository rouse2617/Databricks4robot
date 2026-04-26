# HDF5æ°æ®é

Source: https://io-ai.tech/platform/guides/Pipeline/HDF5/#æºå¨äººæ¨¡åè®­ç»

- æ°æ®æµç¨

- HDF5æ°æ®é

# HDF5æ°æ®é

HDF5ï¼Hierarchical Data Format version 5ï¼æ¯ä¸ç§é«æãçµæ´»çæ°æ®å­å¨æ ¼å¼ï¼å¹¿æ³åºç¨äºå·èº«æºè½é¢åãå¶åå±ç»æï¼ç»åæ°æ®éï¼ä¾¿äºç»ç»åç®¡çå¤æ¨¡æå¤ææ°æ®ï¼æ¯æé«æçæ°æ®è¯»ååè·¨å¹³å°å±äº«ã

## HDF5æ°æ®å¯¼å¥â

å¨å·èº«æºè½é¢åï¼HDF5æä»¶çç»æåå½åæ¹å¼å è®¾å¤ååèå¼ãå¹³å°å·²ééä¸»æµå¤é¨æ°æ®ééç³»ç»ï¼å¦æ¾çµPiperï¼ï¼å¯ç´æ¥å¯¼å¥ç¸å³HDF5æä»¶å¹¶è¿è¡å¯è§åã

å¦ææ¨çHDF5æ°æ®ææªæ¯æï¼è¯·èç³»æä»¬åç¥æ°æ®ç»æï¼æä»¬å°å¿«éå®æééï¼è®©æ¨çå¤æ¨¡ææ°æ®å®ç¾å°å¨å¹³å°ä¸å¯è§åãæ æ³¨åå¯¼åºã

## HDF5æ°æ®å¯¼åºâ

å¹³å°æ¯æå°mcapãbagãhdf5ç­æ ¼å¼çæ°æ®æ æ³¨åå¯¼åºä¸ºHDF5æä»¶ï¼ä¾¿äºåç»­æºå¨å­¦ä¹ æ¨¡åè®­ç»ãæ æ³¨è¿ç¨å°å¨ä½ä¸èªç¶è¯­è¨æä»¤å³èï¼ç¡®ä¿VLAæ¨¡åè½å¤çè§£å¹¶æ§è¡è¯­è¨å½ä»¤ã

æ æ³¨æä½è¯¦è§ï¼æ°æ®æ æ³¨

æ æ³¨å®æåï¼å¯å¨å¯¼åºçé¢éæ©æéæ°æ®å­éè¿è¡å¯¼åºã

- åç»æ°éï¼è®¾ç½®æ¯ä¸ªHDF5æä»¶åå«çåå§æä»¶æ°éãè¥éä¸ä¸å¯¹åºï¼å¯è®¾ä¸º1ã

- æ°æ®å·æ°é¢çï¼æ§å¶æ¯ç§æ°æ®ééæ¬¡æ°ï¼å½±åæä»¶å¤§å°ã

å¯¼åºåï¼æ¨å¯å¨çé¢æ¥çç»æï¼

ä¸è½½åçæ°æ®ï¼

## å¯¼åºæ°æ®ç»æè¯´æâ

å¯¼åºçHDF5æä»¶æåå§æä»¶åç»å½åï¼å¦chunk_001.hdf5ï¼ï¼éç¨æ ç¶ç»æç»ç»æ°æ®ï¼

chunk_001.hdf5

- æ ¹ç»ï¼/ï¼ï¼é¡¶å±ç®å½ã

- å­ç»ï¼å¦/dataã/metaã/dataä¸ææ æ³¨ä»»å¡åºåååä¸ºå­ç»ï¼å¦episode_001ãepisode_002ï¼ã

/data

/meta

- /dataä¸ææ æ³¨ä»»å¡åºåååä¸ºå­ç»ï¼å¦episode_001ãepisode_002ï¼ã

/data

episode_001

episode_002

- æ°æ®éï¼å¦/data/episode_001å±æ§å¼åæ¬taskæ æ³¨çèªç¶è¯­è¨(è±æ)task_zhæ æ³¨çèªç¶è¯­è¨(ä¸­æ)scoreå¨ä½è´¨éè¯åå­å¨çæ°æ®åæ¬actionä¸åçå³èæä»¤ (å¤ç»´æ°ç»)observation.images.*åä¸ªè§è§çåç¼©å¾å (JPEG)observation.stateä¼ æå¨çè§æµå¼ (å¤ç»´æ°ç»)observation.gripperå¤¹çªé­åç¶æçè§æµå¼ (å¤ç»´æ°ç»)

/data/episode_001

- å±æ§å¼åæ¬taskæ æ³¨çèªç¶è¯­è¨(è±æ)task_zhæ æ³¨çèªç¶è¯­è¨(ä¸­æ)scoreå¨ä½è´¨éè¯å

- taskæ æ³¨çèªç¶è¯­è¨(è±æ)

task

- task_zhæ æ³¨çèªç¶è¯­è¨(ä¸­æ)

task_zh

- scoreå¨ä½è´¨éè¯å

score

- å­å¨çæ°æ®åæ¬actionä¸åçå³èæä»¤ (å¤ç»´æ°ç»)observation.images.*åä¸ªè§è§çåç¼©å¾å (JPEG)observation.stateä¼ æå¨çè§æµå¼ (å¤ç»´æ°ç»)observation.gripperå¤¹çªé­åç¶æçè§æµå¼ (å¤ç»´æ°ç»)

- actionä¸åçå³èæä»¤ (å¤ç»´æ°ç»)

action

- observation.images.*åä¸ªè§è§çåç¼©å¾å (JPEG)

observation.images.*

- observation.stateä¼ æå¨çè§æµå¼ (å¤ç»´æ°ç»)

observation.state

- observation.gripperå¤¹çªé­åç¶æçè§æµå¼ (å¤ç»´æ°ç»)

observation.gripper

ç¤ºä¾ç»æå¦ä¸ï¼

HDF5 "./chunk_001.hdf5" {FILE_CONTENTS {group      /group      /datagroup      /data/episode_001dataset    /data/episode_001/actiondataset    /data/episode_001/observation.gripperdataset    /data/episode_001/observation.images.camera_01dataset    /data/episode_001/observation.images.camera_02dataset    /data/episode_001/observation.images.camera_03dataset    /data/episode_001/observation.images.camera_04dataset    /data/episode_001/observation.stategroup      /data/episode_002dataset    /data/episode_002/actiondataset    /data/episode_002/observation.gripperdataset    /data/episode_002/observation.images.camera_01dataset    /data/episode_002/observation.images.camera_02dataset    /data/episode_002/observation.images.camera_03dataset    /data/episode_002/observation.images.camera_04dataset    /data/episode_002/observation.state......group      /meta}}

HDF5 "./chunk_001.hdf5" {FILE_CONTENTS {group      /group      /datagroup      /data/episode_001dataset    /data/episode_001/actiondataset    /data/episode_001/observation.gripperdataset    /data/episode_001/observation.images.camera_01dataset    /data/episode_001/observation.images.camera_02dataset    /data/episode_001/observation.images.camera_03dataset    /data/episode_001/observation.images.camera_04dataset    /data/episode_001/observation.stategroup      /data/episode_002dataset    /data/episode_002/actiondataset    /data/episode_002/observation.gripperdataset    /data/episode_002/observation.images.camera_01dataset    /data/episode_002/observation.images.camera_02dataset    /data/episode_002/observation.images.camera_03dataset    /data/episode_002/observation.images.camera_04dataset    /data/episode_002/observation.state......group      /meta}}

## HDF5æä»¶è¯»åç¤ºä¾â

æ¨èä½¿ç¨ Python çh5pyåºè¯»ååæä½HDF5æä»¶ãåºæ¬ç¨æ³å¦ä¸ï¼

importh5py# ä»¥åªè¯»æ¨¡å¼æå¼ HDF5 æä»¶withh5py.File('chunk_001.hdf5','r')asf:# æ¥çé¡¶å±ç»print("é¡¶å±ç»ï¼",list(f.keys()))# è®¿é® /data/episode_001 ç»ä¸çæ°æ®éepisode_001=f['/data/episode_001']print("episode_001 ä¸çæ°æ®éï¼",list(episode_001.keys()))# è¯»å action æ°æ®éaction_data=episode_001['action'][:]print("action æ°æ®ï¼",action_data)

importh5py# ä»¥åªè¯»æ¨¡å¼æå¼ HDF5 æä»¶withh5py.File('chunk_001.hdf5','r')asf:# æ¥çé¡¶å±ç»print("é¡¶å±ç»ï¼",list(f.keys()))# è®¿é® /data/episode_001 ç»ä¸çæ°æ®éepisode_001=f['/data/episode_001']print("episode_001 ä¸çæ°æ®éï¼",list(episode_001.keys()))# è¯»å action æ°æ®éaction_data=episode_001['action'][:]print("action æ°æ®ï¼",action_data)

## åºç¨åºæ¯ä¸ä¼å¿â

HDF5æ ¼å¼å¨å·èº«æºè½é¢åå·æä»¥ä¸ä¼å¿ï¼

- æ¯æå¤§è§æ¨¡å¤æ¨¡ææ°æ®å­å¨ï¼å¦é«åè¾¨çå¾åãä¼ æå¨æ°æ®ç­ï¼

- åç½®æ°æ®åç¼©ï¼èçå­å¨ç©ºé´

- è·¨å¹³å°å¼å®¹ï¼ä¾¿äºæ°æ®å±äº«åè¿ç§»

- çµæ´»çæ°æ®å±æ¬¡ç»æï¼éåå¤æä»»å¡åå¤æ ·åæ°æ®ç®¡ç

åçè®¾è®¡HDF5ç»æï¼ç»åå¹³å°å·¥å·ï¼è½é«æç®¡çåå¤çå·èº«æºè½ç¸å³çå¤ææ°æ®ï¼å©åç§å­¦ç ç©¶åæ¨¡åè®­ç»ã

## æºå¨äººæ¨¡åè®­ç»â

å¯¼åºçHDF5æ°æ®å¯ä»¥ç´æ¥ç¨äºåç§æºå¨äººå­¦ä¹ æ¨¡åçè®­ç»ï¼åæ¬æ¨¡ä»¿å­¦ä¹ ãå¼ºåå­¦ä¹ åè§è§-è¯­è¨-å¨ä½ï¼VLAï¼æ¨¡åç­ã

è¯¦ç»çè®­ç»æ¹æ³åä»£ç ç¤ºä¾è¯·åèï¼HDF5æ°æ®ç¨äºæºå¨äººæ¨¡åè®­ç»

## Images on this page

- ![éæ©è¦å¯¼åºçæ°æ®](../images/platform_assets_images_selected_4b051c668a6876c96aee5c699ab4cc25_webp.webp)
- ![æ¥çå¯¼åºç»æ](../images/platform_assets_images_success_4d398167e06f3d4de3b60f2249ce724f_webp.webp)
