# æ°æ®æ ¼å¼

Source: https://io-ai.tech/platform/guides/Structure/#lerobotæ ¼å¼

- æ°æ®æ ¼å¼

# æ°æ®æ ¼å¼

è¾æ¬§æ°æ®å¹³å°æ¯æçµæ´»çæ°æ®æ ¼å¼ï¼å¯ä»¥èªå®ä¹æ°æ®å¯è§åæ¨¡æ¿ã

è¿éä»¥è¾æ¬§æ°æ®ééäº§åééçæ°æ®æ ¼å¼ä¸ºä¾ï¼

## äººç±»æ°æ®æ ¼å¼â

### æä»¶ç»æâ

f"{date}_{project}_{scene}_{task}_{staff_id}_{timestamp}"âââ align_result.csv # æ¶é´æ³å¯¹é½è¡¨æ ¼âââ annotation.json  # æ æ³¨æ°æ®âââ config           # ç¸æºåä¼ æå¨éç½®âÂ Â  âââ calib_data.ymlâÂ Â  âââ depth_to_rgb.ymlâÂ Â  âââ mocap_main.ymlâÂ Â  âââ orbbec_depth.ymlâÂ Â  âââ orbbec_rgb.ymlâÂ Â  âââ pose_calib.ymlâââ data.mcap        # å¤æ¨¡ææ°æ®

f"{date}_{project}_{scene}_{task}_{staff_id}_{timestamp}"âââ align_result.csv # æ¶é´æ³å¯¹é½è¡¨æ ¼âââ annotation.json  # æ æ³¨æ°æ®âââ config           # ç¸æºåä¼ æå¨éç½®âÂ Â  âââ calib_data.ymlâÂ Â  âââ depth_to_rgb.ymlâÂ Â  âââ mocap_main.ymlâÂ Â  âââ orbbec_depth.ymlâÂ Â  âââ orbbec_rgb.ymlâÂ Â  âââ pose_calib.ymlâââ data.mcap        # å¤æ¨¡ææ°æ®

### å¤æ¨¡ææ°æ®â

library:   mcap go v1.7.0profile:   ros1messages:  45200duration:  1m5.625866496sstart:     2025-01-15T18:09:29.628202496+08:00 (1736935769.628202496)end:       2025-01-15T18:10:35.254068992+08:00 (1736935835.254068992)compression:zstd: [764/764 chunks] [6.13 GiB/3.84 GiB (37.39%)] [59.87 MiB/sec]channels:(1)  /rgbd/color/image_raw/compressed                  1970 msgs (30.02 Hz)   : sensor_msgs/CompressedImage [ros1msg](2)  /joint_states                                     1970 msgs (30.02 Hz)   : sensor_msgs/JointState [ros1msg](3)  /claws_r_hand                                     1970 msgs (30.02 Hz)   : io_msgs/claws_angle [ros1msg](4)  /global_localization                              1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg](5)  /robot_description                                   1 msgs              : std_msgs/String [ros1msg](6)  /ee_visualization                                 1970 msgs (30.02 Hz)   : sensor_msgs/CompressedImage [ros1msg](7)  /rgbd/depth/image_raw                             1970 msgs (30.02 Hz)   : sensor_msgs/Image [ros1msg](8)  /colorized_depth                                  1970 msgs (30.02 Hz)   : sensor_msgs/CompressedImage [ros1msg](9)  /claws_l_hand                                     1970 msgs (30.02 Hz)   : io_msgs/claws_angle [ros1msg](10) /claws_touch_data                                 1970 msgs (30.02 Hz)   : io_msgs/squashed_touch [ros1msg](11) /touch_visualization                              1970 msgs (30.02 Hz)   : sensor_msgs/CompressedImage [ros1msg](12) /mocap/sensor_data                                1970 msgs (30.02 Hz)   : io_msgs/squashed_mocap_data [ros1msg](13) /mocap/ros_tf                                     1970 msgs (30.02 Hz)   : tf2_msgs/TFMessage [ros1msg](14) /left_ee_pose                                     1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg](15) /right_ee_pose                                    1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg](16) /usb_cam_left/mjpeg_raw/compressed                1960 msgs (29.87 Hz)   : sensor_msgs/CompressedImage [ros1msg](17) /usb_cam_right/mjpeg_raw/compressed               1946 msgs (29.65 Hz)   : sensor_msgs/CompressedImage [ros1msg](18) /usb_cam_fisheye/mjpeg_raw/compressed             1957 msgs (29.82 Hz)   : sensor_msgs/CompressedImage [ros1msg](19) /realsense_left_hand/depth/image_rect_raw         1961 msgs (29.88 Hz)   : sensor_msgs/Image [ros1msg](20) /realsense_left_hand/color/image_raw/compressed   1961 msgs (29.88 Hz)   : sensor_msgs/CompressedImage [ros1msg](21) /realsense_right_hand/depth/image_rect_raw        1947 msgs (29.67 Hz)   : sensor_msgs/Image [ros1msg](22) /realsense_right_hand/color/image_raw/compressed  1947 msgs (29.67 Hz)   : sensor_msgs/CompressedImage [ros1msg](23) /world_left_ee_pose                               1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg](24) /world_right_ee_pose                              1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg]channels: 24attachments: 0metadata: 0

library:   mcap go v1.7.0profile:   ros1messages:  45200duration:  1m5.625866496sstart:     2025-01-15T18:09:29.628202496+08:00 (1736935769.628202496)end:       2025-01-15T18:10:35.254068992+08:00 (1736935835.254068992)compression:zstd: [764/764 chunks] [6.13 GiB/3.84 GiB (37.39%)] [59.87 MiB/sec]channels:(1)  /rgbd/color/image_raw/compressed                  1970 msgs (30.02 Hz)   : sensor_msgs/CompressedImage [ros1msg](2)  /joint_states                                     1970 msgs (30.02 Hz)   : sensor_msgs/JointState [ros1msg](3)  /claws_r_hand                                     1970 msgs (30.02 Hz)   : io_msgs/claws_angle [ros1msg](4)  /global_localization                              1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg](5)  /robot_description                                   1 msgs              : std_msgs/String [ros1msg](6)  /ee_visualization                                 1970 msgs (30.02 Hz)   : sensor_msgs/CompressedImage [ros1msg](7)  /rgbd/depth/image_raw                             1970 msgs (30.02 Hz)   : sensor_msgs/Image [ros1msg](8)  /colorized_depth                                  1970 msgs (30.02 Hz)   : sensor_msgs/CompressedImage [ros1msg](9)  /claws_l_hand                                     1970 msgs (30.02 Hz)   : io_msgs/claws_angle [ros1msg](10) /claws_touch_data                                 1970 msgs (30.02 Hz)   : io_msgs/squashed_touch [ros1msg](11) /touch_visualization                              1970 msgs (30.02 Hz)   : sensor_msgs/CompressedImage [ros1msg](12) /mocap/sensor_data                                1970 msgs (30.02 Hz)   : io_msgs/squashed_mocap_data [ros1msg](13) /mocap/ros_tf                                     1970 msgs (30.02 Hz)   : tf2_msgs/TFMessage [ros1msg](14) /left_ee_pose                                     1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg](15) /right_ee_pose                                    1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg](16) /usb_cam_left/mjpeg_raw/compressed                1960 msgs (29.87 Hz)   : sensor_msgs/CompressedImage [ros1msg](17) /usb_cam_right/mjpeg_raw/compressed               1946 msgs (29.65 Hz)   : sensor_msgs/CompressedImage [ros1msg](18) /usb_cam_fisheye/mjpeg_raw/compressed             1957 msgs (29.82 Hz)   : sensor_msgs/CompressedImage [ros1msg](19) /realsense_left_hand/depth/image_rect_raw         1961 msgs (29.88 Hz)   : sensor_msgs/Image [ros1msg](20) /realsense_left_hand/color/image_raw/compressed   1961 msgs (29.88 Hz)   : sensor_msgs/CompressedImage [ros1msg](21) /realsense_right_hand/depth/image_rect_raw        1947 msgs (29.67 Hz)   : sensor_msgs/Image [ros1msg](22) /realsense_right_hand/color/image_raw/compressed  1947 msgs (29.67 Hz)   : sensor_msgs/CompressedImage [ros1msg](23) /world_left_ee_pose                               1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg](24) /world_right_ee_pose                              1970 msgs (30.02 Hz)   : geometry_msgs/PoseStamped [ros1msg]channels: 24attachments: 0metadata: 0

Topicåç§°

æ°æ®å«ä¹

/mocap/sensor_data

åºäºå¨ä½ææçå³èéåº¦ãå éåº¦ãè§éåº¦ã æè½¬è§åº¦åä¼ æå¨æ°æ®

/mocap/ros_tf

åºäºå¨ä½ææçææå³èçTF

/joint_states

åºäºå¨ä½ææçææå³èçJointState

/right_ee_pose

ä¸»å¤´é¨ç¸æºåæ ç³»ä¸çå³å¤¹çªä½å§¿

/left_ee_pose

ä¸»å¤´é¨ç¸æºåæ ç³»ä¸çå·¦å¤¹çªä½å§¿

/claws_l_hand

å·¦å¤¹çªé­åç¨åº¦

/claws_r_hand

å³å¤¹çªé­åç¨åº¦

/claws_touch_data

å¤¹çªè§¦è§æ°æ®ï¼åå«ä¸¤ä¸ªæ¶æ¯ï¼æ¯ä¸ªæ¶æ¯çframe_idè¡¨ç¤ºå·¦æå³å¤¹çªï¼dataçååä¸ªå¼ææï¼

/realsense_left_hand/color/image_raw/compressed

å·¦å¤¹çªç¸æºçRGBå¾å

/realsense_left_hand/depth/image_rect_raw

å·¦å¤¹çªç¸æºçæ·±åº¦å¾å

/realsense_right_hand/color/image_raw/compressed

å³å¤¹çªç¸æºçRGBå¾å

/realsense_right_hand/depth/image_rect_raw

å³å¤¹çªç¸æºçæ·±åº¦å¾å

/rgbd/color/image_raw/compressed

ä¸»å¤´é¨ç¸æºçRGBå¾å

/rgbd/depth/image_raw

ä¸»å¤´é¨ç¸æºçæ·±åº¦å¾å

/colorized_depth

ä¸»å¤´é¨ç¸æºçå½©è²æ·±åº¦å¾å

/usb_cam_fisheye/mjpeg_raw/compressed

ä¸»å¤´é¨é±¼ç¼ç¸æºçRGBå¾å

/usb_cam_left/mjpeg_raw/compressed

ä¸»å¤´é¨å·¦åç®ç¸æºçRGBå¾å

/usb_cam_right/mjpeg_raw/compressed

ä¸»å¤´é¨å³åç®ç¸æºçRGBå¾å

/ee_visualization

ä¸»å¤´é¨ç¸æºRGBå¾åä¸­çæ«ç«¯æ§è¡å¨ä½å§¿å¯è§å

/touch_visualization

å¤¹çªè§¦è§æ°æ®å¯è§å

/robot_description

å¨ä½ææURDF

/global_localization

ä¸»å¤´é¨ç¸æºå¨ä¸çåæ ç³»ä¸­çä½å§¿

/world_left_ee_pose

å·¦å¤¹çªå¨ä¸çåæ ç³»ä¸­çä½å§¿

/world_right_ee_pose

  å³å¤¹çªå¨ä¸çåæ ç³»ä¸­çä½å§¿

å¦ææ¯äººç©¿æ´çè§¦è§æå¥ééçæ°æ®ï¼ä¼å¢å è§¦è§æ°å­ä¿¡å·éµåTopicï¼

/mocap/touch_data 57 msgs (30.25 Hz): io_msgs/squashed_touc [ros1msg]

### èªç¶è¯­è¨æ æ³¨æ°æ®â

{"belong_to":"20250115_InnerTest_PublicArea_TableClearing_szk_180926","mocap_offset":[],"object_set":["paper cup","placemat","trash can","napkin","plate","dinner knife","tableware storage box","wine glass","dinner fork"],"scene":"PublicArea","skill_set":["pick {A} from {B}","toss {A} into {B}","place {A} on {B}"],"subtasks":[{"skill":"pick {A} from {B}","description":"pick the paper cup from the placemat with the left gripper","description_zh":"å·¦å¤¹çª ä» é¤å« æ¡èµ· çº¸æ¯","end_frame_id":227,"end_timestamp":"1736935777206000000","sequence_id":1,"start_frame_id":159,"start_timestamp":"1736935774906000000","comment":"","attempts":"success"},{"skill":"toss {A} into {B}","description":"toss the paper cup into the trash can with the left gripper","description_zh":"å·¦å¤¹çª æçº¸æ¯è¿åå¾æ¡¶","end_frame_id":318,"end_timestamp":"1736935780244000000","sequence_id":2,"start_frame_id":231,"start_timestamp":"1736935777306000000","comment":"","attempts":"success"},...],"tag_set":[],"task_description":"20250115_InnerTest_PublicArea_TableClearing_szk_180926"}

{"belong_to":"20250115_InnerTest_PublicArea_TableClearing_szk_180926","mocap_offset":[],"object_set":["paper cup","placemat","trash can","napkin","plate","dinner knife","tableware storage box","wine glass","dinner fork"],"scene":"PublicArea","skill_set":["pick {A} from {B}","toss {A} into {B}","place {A} on {B}"],"subtasks":[{"skill":"pick {A} from {B}","description":"pick the paper cup from the placemat with the left gripper","description_zh":"å·¦å¤¹çª ä» é¤å« æ¡èµ· çº¸æ¯","end_frame_id":227,"end_timestamp":"1736935777206000000","sequence_id":1,"start_frame_id":159,"start_timestamp":"1736935774906000000","comment":"","attempts":"success"},{"skill":"toss {A} into {B}","description":"toss the paper cup into the trash can with the left gripper","description_zh":"å·¦å¤¹çª æçº¸æ¯è¿åå¾æ¡¶","end_frame_id":318,"end_timestamp":"1736935780244000000","sequence_id":2,"start_frame_id":231,"start_timestamp":"1736935777306000000","comment":"","attempts":"success"},...],"tag_set":[],"task_description":"20250115_InnerTest_PublicArea_TableClearing_szk_180926"}

## é¥æä½æºå¨äººæ°æ®æ ¼å¼â

### æä»¶ç»æâ

f"{robot_name}_{date}_{timestamp}_{sequence_id}"âââ RM_AIDAL_250124_172033_0.mcapâââ RM_AIDAL_250124_172033_0.jsonâââ RM_AIDAL_250126_093648_0.metadata.yaml

f"{robot_name}_{date}_{timestamp}_{sequence_id}"âââ RM_AIDAL_250124_172033_0.mcapâââ RM_AIDAL_250124_172033_0.jsonâââ RM_AIDAL_250126_093648_0.metadata.yaml

### å¤æ¨¡ææ°æ®â

Files:             RM_AIDAL_250126_091041_0.mcapBag size:          443.3 MiBStorage id:        mcapDuration:          100.052164792sStart:             Jan 24 2025 21:37:32.526605552 (1737725852.526605552)End:               Jan 24 2025 21:39:12.578770344 (1737725952.578770344)Messages:          62116Topic information: Topic: /camera_01/color/image_raw/compressed | Type: sensor_msgs/msg/CompressedImage | Count: 3000 | Serialization Format: cdrTopic: /camera_02/color/image_raw/compressed | Type: sensor_msgs/msg/CompressedImage | Count: 3000 | Serialization Format: cdrTopic: /camera_03/color/image_raw/compressed | Type: sensor_msgs/msg/CompressedImage | Count: 3000 | Serialization Format: cdrTopic: io_teleop/joint_states | Type: sensor_msgs/msg/JointState | Count: 1529 | Serialization Format: cdrTopic: io_teleop/joint_cmd | Type: sensor_msgs/msg/JointState | Count: 10009 | Serialization Format: cdrTopic: io_teleop/target_ee_poses | Type: geometry_msgs/msg/PoseArray | Count: 10014 | Serialization Format: cdrTopic: io_teleop/target_base_move | Type: std_msgs/msg/Float64MultiArray | Count: 10010 | Serialization Format: cdrTopic: io_teleop/target_gripper_status | Type: sensor_msgs/msg/JointState | Count: 10012 | Serialization Format: cdrTopic: io_teleop/target_joint_from_vr | Type: sensor_msgs/msg/JointState | Count: 10012 | Serialization Format: cdrTopic: /robot_description | Type: std_msgs/msg/String | Count: 1 | Serialization Format: cdrTopic: /tf | Type: tf2_msgs/msg/TFMessage | Count: 1529 | Serialization Format: cdr

Files:             RM_AIDAL_250126_091041_0.mcapBag size:          443.3 MiBStorage id:        mcapDuration:          100.052164792sStart:             Jan 24 2025 21:37:32.526605552 (1737725852.526605552)End:               Jan 24 2025 21:39:12.578770344 (1737725952.578770344)Messages:          62116Topic information: Topic: /camera_01/color/image_raw/compressed | Type: sensor_msgs/msg/CompressedImage | Count: 3000 | Serialization Format: cdrTopic: /camera_02/color/image_raw/compressed | Type: sensor_msgs/msg/CompressedImage | Count: 3000 | Serialization Format: cdrTopic: /camera_03/color/image_raw/compressed | Type: sensor_msgs/msg/CompressedImage | Count: 3000 | Serialization Format: cdrTopic: io_teleop/joint_states | Type: sensor_msgs/msg/JointState | Count: 1529 | Serialization Format: cdrTopic: io_teleop/joint_cmd | Type: sensor_msgs/msg/JointState | Count: 10009 | Serialization Format: cdrTopic: io_teleop/target_ee_poses | Type: geometry_msgs/msg/PoseArray | Count: 10014 | Serialization Format: cdrTopic: io_teleop/target_base_move | Type: std_msgs/msg/Float64MultiArray | Count: 10010 | Serialization Format: cdrTopic: io_teleop/target_gripper_status | Type: sensor_msgs/msg/JointState | Count: 10012 | Serialization Format: cdrTopic: io_teleop/target_joint_from_vr | Type: sensor_msgs/msg/JointState | Count: 10012 | Serialization Format: cdrTopic: /robot_description | Type: std_msgs/msg/String | Count: 1 | Serialization Format: cdrTopic: /tf | Type: tf2_msgs/msg/TFMessage | Count: 1529 | Serialization Format: cdr

Topicåç§°

æ°æ®å«ä¹

/camera_01/color/image_raw/compressed

ä¸»ç¸æºçRGBå¾å

/camera_02/color/image_raw/compressed

å·¦ç¸æºçRGBå¾å

/camera_03/color/image_raw/compressed

å³ç¸æºçRGBå¾å

io_teleop/joint_states

å³èç¶æ

io_teleop/joint_cmd

å³èå½ä»¤

io_teleop/target_ee_poses

æ«ç«¯æ§è¡å¨ç®æ ä½å§¿

io_teleop/target_base_move

åºåº§ç§»å¨ç®æ 

io_teleop/target_gripper_status

å¤¹çªç¶æç®æ 

io_teleop/target_joint_from_vr

VRè®¾å¤çå³èç®æ 

/robot_description

æºå¨äººURDFæè¿°

/tf

TFç©ºé´ä½å§¿åæ¢ä¿¡æ¯

### èªç¶è¯­è¨æ æ³¨æ°æ®â

{"belong_to":"RM_AIDAL_250126_091041_0","mocap_offset":[],"object_set":["lemon candy","plate","pistachios"],"scene":"250126","skill_set":["place {A} on {B}"],"subtasks":[{"skill":"place {A} on {B}","objecta":"lemon candy","objectb":"plate","options":["leftHand"],"description":"place the lemon candy on the plate with the left hand","end_timestamp":"1737725886915000000","sequence_id":1,"start_timestamp":"1737725880757000000","comment":"","attempts":"success"},{"skill":"place {A} on {B}","objecta":"pistachios","objectb":"plate","options":["rightHand"],"description":"place the pistachios on the plate with the right hand","end_timestamp":"1737725950745000000","sequence_id":2,"start_timestamp":"1737725941657000000","comment":"","attempts":"success"}],"tag_set":[],"task_description":"20250205_RM_ItemPacking_zhouxw"}

{"belong_to":"RM_AIDAL_250126_091041_0","mocap_offset":[],"object_set":["lemon candy","plate","pistachios"],"scene":"250126","skill_set":["place {A} on {B}"],"subtasks":[{"skill":"place {A} on {B}","objecta":"lemon candy","objectb":"plate","options":["leftHand"],"description":"place the lemon candy on the plate with the left hand","end_timestamp":"1737725886915000000","sequence_id":1,"start_timestamp":"1737725880757000000","comment":"","attempts":"success"},{"skill":"place {A} on {B}","objecta":"pistachios","objectb":"plate","options":["rightHand"],"description":"place the pistachios on the plate with the right hand","end_timestamp":"1737725950745000000","sequence_id":2,"start_timestamp":"1737725941657000000","comment":"","attempts":"success"}],"tag_set":[],"task_description":"20250205_RM_ItemPacking_zhouxw"}

## æ¨¡åè®­ç»æ°æ®â

æä»¬æä¾å°ä¸è¿°mcapåjsonæ°æ®è½¬æ¢æpythonå¯ä»¥è§£æçæ°æ®ï¼ç¨äºç´æ¥æå¥å¤§æ¨¡åçæ°æ®è®­ç»ã

### HDF5æ ¼å¼â

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

### LeRobotæ ¼å¼â

å¯åèæä»¬çåºç¡æ ·ä¾æ°æ®ï¼https://huggingface.co/datasets/io-ai-data/DesktopCleanup_RM_AIDAL_demo

{"codebase_version":"v2.1","robot_type":"custom_arm","total_episodes":20,"total_frames":5134,"total_tasks":20,"total_videos":0,"total_chunks":1,"chunks_size":1000,"fps":30,"splits":{"train":"0:20"},"data_path":"data/chunk-{episode_chunk:03d}/episode_{episode_index:06d}.parquet","video_path":"videos/chunk-{episode_chunk:03d}/{video_key}/episode_{episode_index:06d}.mp4","features":{"observation.images.camera_01":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_02":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_03":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_04":{"dtype":"image","shape":[480,640,3]},"observation.state":{"dtype":"float64","shape":[37],"names":["r_joint1","r_joint2","r_joint3","r_joint4","r_joint5","r_joint6","l_joint1","l_joint2","l_joint3","l_joint4","l_joint5","l_joint6","R_thumb_MCP_joint1","R_thumb_MCP_joint2","R_thumb_PIP_joint","R_thumb_DIP_joint","R_index_MCP_joint","R_index_DIP_joint","R_middle_MCP_joint","R_middle_DIP_joint","R_ring_MCP_joint","R_ring_DIP_joint","R_pinky_MCP_joint","R_pinky_DIP_joint","L_thumb_MCP_joint1","L_thumb_MCP_joint2","L_thumb_PIP_joint","L_thumb_DIP_joint","L_index_MCP_joint","L_index_DIP_joint","L_middle_MCP_joint","L_middle_DIP_joint","L_ring_MCP_joint","L_ring_DIP_joint","L_pinky_MCP_joint","L_pinky_DIP_joint","platform_joint"]},"action":{"dtype":"float64","shape":[12],"names":["l_joint1","l_joint2","l_joint3","l_joint4","l_joint5","l_joint6","r_joint1","r_joint2","r_joint3","r_joint4","r_joint5","r_joint6"]},"observation.gripper":{"dtype":"float64","shape":[2],"names":["right_gripper","left_gripper"]},"timestamp":{"dtype":"float32","shape":[1],"names":null},"frame_index":{"dtype":"int64","shape":[1],"names":null},"episode_index":{"dtype":"int64","shape":[1],"names":null},"index":{"dtype":"int64","shape":[1],"names":null},"task_index":{"dtype":"int64","shape":[1],"names":null}}}

{"codebase_version":"v2.1","robot_type":"custom_arm","total_episodes":20,"total_frames":5134,"total_tasks":20,"total_videos":0,"total_chunks":1,"chunks_size":1000,"fps":30,"splits":{"train":"0:20"},"data_path":"data/chunk-{episode_chunk:03d}/episode_{episode_index:06d}.parquet","video_path":"videos/chunk-{episode_chunk:03d}/{video_key}/episode_{episode_index:06d}.mp4","features":{"observation.images.camera_01":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_02":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_03":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_04":{"dtype":"image","shape":[480,640,3]},"observation.state":{"dtype":"float64","shape":[37],"names":["r_joint1","r_joint2","r_joint3","r_joint4","r_joint5","r_joint6","l_joint1","l_joint2","l_joint3","l_joint4","l_joint5","l_joint6","R_thumb_MCP_joint1","R_thumb_MCP_joint2","R_thumb_PIP_joint","R_thumb_DIP_joint","R_index_MCP_joint","R_index_DIP_joint","R_middle_MCP_joint","R_middle_DIP_joint","R_ring_MCP_joint","R_ring_DIP_joint","R_pinky_MCP_joint","R_pinky_DIP_joint","L_thumb_MCP_joint1","L_thumb_MCP_joint2","L_thumb_PIP_joint","L_thumb_DIP_joint","L_index_MCP_joint","L_index_DIP_joint","L_middle_MCP_joint","L_middle_DIP_joint","L_ring_MCP_joint","L_ring_DIP_joint","L_pinky_MCP_joint","L_pinky_DIP_joint","platform_joint"]},"action":{"dtype":"float64","shape":[12],"names":["l_joint1","l_joint2","l_joint3","l_joint4","l_joint5","l_joint6","r_joint1","r_joint2","r_joint3","r_joint4","r_joint5","r_joint6"]},"observation.gripper":{"dtype":"float64","shape":[2],"names":["right_gripper","left_gripper"]},"timestamp":{"dtype":"float32","shape":[1],"names":null},"frame_index":{"dtype":"int64","shape":[1],"names":null},"episode_index":{"dtype":"int64","shape":[1],"names":null},"index":{"dtype":"int64","shape":[1],"names":null},"task_index":{"dtype":"int64","shape":[1],"names":null}}}