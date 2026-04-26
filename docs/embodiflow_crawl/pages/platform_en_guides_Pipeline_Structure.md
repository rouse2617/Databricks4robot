# Data Formats

Source: https://io-ai.tech/platform/en/guides/Pipeline/Structure/#lerobot-format

- Data Pipeline

- Data Formats

# Data Formats

EmbodiFlow Platform is designed for general robot data management, withRobot Operating System (ROS)as the benchmark for unified robot data management.

- Data Import: Supports automatic conversion of non-ROS standard data from systems like AgiBot and AgileX into ROS standard format for unified management.

- Data Visualization: Built-in visualization models for over 30 mainstream robots, allowing smooth playback of all formats including 3D animations and 2D images.

- Data Export: Supports one-click export to standard HDF5/LeRobot data formats, with adaptive joint and image mapping based on original data, ready for model training.

### Data Flow and Format Relationshipâ

The diagram below summarizes how human data and teleoperation robot data are unified under ROS/MCAP management and exported for training.

## Table of Contentsâ

- Human Data FormatFile StructureMultimodal DataNatural Language Annotation

- File Structure

- Multimodal Data

- Natural Language Annotation

- Teleoperation Robot Data FormatTeleoperation File StructureTeleoperation Multimodal DataTeleoperation Annotation Data

- Teleoperation File Structure

- Teleoperation Multimodal Data

- Teleoperation Annotation Data

- Exporting Model Training DataHDF5 FormatLeRobot Format

- HDF5 Format

- LeRobot Format

## Human Data Formatâ

Human data collection is primarily used to record the operator's actions and interaction processes, containing multimodal sensor data.

### File Structureâ

Each collection task generates a folder named with a timestamp:

f"{date}_{project}_{scene}_{task}_{staff_id}_{timestamp}"âââ align_result.csv    # Timestamp alignment tableâââ annotation.json     # Annotation dataâââ config/            # Camera and sensor configurationâ   âââ calib_data.ymlâ   âââ depth_to_rgb.ymlâ   âââ mocap_main.ymlâ   âââ orbbec_depth.ymlâ   âââ orbbec_rgb.ymlâ   âââ pose_calib.ymlâââ data.mcap          # Multimodal data package

f"{date}_{project}_{scene}_{task}_{staff_id}_{timestamp}"âââ align_result.csv    # Timestamp alignment tableâââ annotation.json     # Annotation dataâââ config/            # Camera and sensor configurationâ   âââ calib_data.ymlâ   âââ depth_to_rgb.ymlâ   âââ mocap_main.ymlâ   âââ orbbec_depth.ymlâ   âââ orbbec_rgb.ymlâ   âââ pose_calib.ymlâââ data.mcap          # Multimodal data package

### Multimodal Dataâ

Thedata.mcapfile contains all synchronized sensor data, stored in MCAP format.

data.mcap

Main Topic List:

Topic Name

Data Type

Description

/mocap/sensor_data

/mocap/sensor_data

io_msgs/squashed_mocap_data

io_msgs/squashed_mocap_data

Joint velocity, acceleration, angular velocity, rotation angle, and sensor data from motion capture

/mocap/ros_tf

/mocap/ros_tf

tf2_msgs/TFMessage

tf2_msgs/TFMessage

TF transforms for all joints based on motion capture

/joint_states

/joint_states

sensor_msgs/JointState

sensor_msgs/JointState

JointStates for all joints based on motion capture

/rgbd/color/image_raw/compressed

/rgbd/color/image_raw/compressed

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

RGB image from the main head camera

/rgbd/depth/image_raw

/rgbd/depth/image_raw

sensor_msgs/Image

sensor_msgs/Image

Depth image from the main head camera

/colorized_depth

/colorized_depth

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

Colorized depth image from the main head camera

/left_ee_pose

/left_ee_pose

geometry_msgs/PoseStamped

geometry_msgs/PoseStamped

Left gripper pose in the main head camera coordinate system

/right_ee_pose

/right_ee_pose

geometry_msgs/PoseStamped

geometry_msgs/PoseStamped

Right gripper pose in the main head camera coordinate system

/claws_l_hand

/claws_l_hand

io_msgs/claws_angle

io_msgs/claws_angle

Left gripper closure degree

/claws_r_hand

/claws_r_hand

io_msgs/claws_angle

io_msgs/claws_angle

Right gripper closure degree

/claws_touch_data

/claws_touch_data

io_msgs/squashed_touch

io_msgs/squashed_touch

Gripper tactile data

/realsense_left_hand/color/image_raw/compressed

/realsense_left_hand/color/image_raw/compressed

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

RGB image from the left gripper camera

/realsense_left_hand/depth/image_rect_raw

/realsense_left_hand/depth/image_rect_raw

sensor_msgs/Image

sensor_msgs/Image

Depth image from the left gripper camera

/realsense_right_hand/color/image_raw/compressed

/realsense_right_hand/color/image_raw/compressed

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

RGB image from the right gripper camera

/realsense_right_hand/depth/image_rect_raw

/realsense_right_hand/depth/image_rect_raw

sensor_msgs/Image

sensor_msgs/Image

Depth image from the right gripper camera

/usb_cam_fisheye/mjpeg_raw/compressed

/usb_cam_fisheye/mjpeg_raw/compressed

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

RGB image from the main head fisheye camera

/usb_cam_left/mjpeg_raw/compressed

/usb_cam_left/mjpeg_raw/compressed

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

RGB image from the main head left monocular camera

/usb_cam_right/mjpeg_raw/compressed

/usb_cam_right/mjpeg_raw/compressed

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

RGB image from the main head right monocular camera

/ee_visualization

/ee_visualization

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

End-effector pose visualization in the main head camera RGB image

/touch_visualization

/touch_visualization

sensor_msgs/CompressedImage

sensor_msgs/CompressedImage

Gripper tactile data visualization

/robot_description

/robot_description

std_msgs/String

std_msgs/String

Motion capture URDF

/global_localization

/global_localization

geometry_msgs/PoseStamped

geometry_msgs/PoseStamped

Main head camera pose in the world coordinate system

/world_left_ee_pose

/world_left_ee_pose

geometry_msgs/PoseStamped

geometry_msgs/PoseStamped

Left gripper pose in the world coordinate system

/world_right_ee_pose

/world_right_ee_pose

geometry_msgs/PoseStamped

geometry_msgs/PoseStamped

Right gripper pose in the world coordinate system

Camera Data:

- Main Head RGBD Camera: Color + Depth images

- Left/Right Gripper Camera: RealSense RGBD

- Fisheye Camera: Panoramic view

- Left/Right Monocular Camera: Stereo vision

Note:If tactile gloves are used, an additional/mocap/touch_datatopic will be added.

/mocap/touch_data

library:mcap go v1.7.0profile:ros1messages:45200duration:1m5.625866496sstart:2025-01-15T18:09:29.628202496+08:00 (1736935769.628202496)end:2025-01-15T18:10:35.254068992+08:00 (1736935835.254068992)compression:zstd:[764/764 chunks][6.13 GiB/3.84 GiB (37.39%)][59.87 MiB/sec]channels:(1)  /rgbd/color/image_raw/compressed                  1970 msgs (30.02 Hz):sensor_msgs/CompressedImage[ros1msg](2)  /joint_states                                     1970 msgs (30.02 Hz):sensor_msgs/JointState[ros1msg](3)  /claws_r_hand                                     1970 msgs (30.02 Hz):io_msgs/claws_angle[ros1msg](4)  /global_localization                              1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg](5)  /robot_description                                   1 msgs:std_msgs/String[ros1msg](6)  /ee_visualization                                 1970 msgs (30.02 Hz):sensor_msgs/CompressedImage[ros1msg](7)  /rgbd/depth/image_raw                             1970 msgs (30.02 Hz):sensor_msgs/Image[ros1msg](8)  /colorized_depth                                  1970 msgs (30.02 Hz):sensor_msgs/CompressedImage[ros1msg](9)  /claws_l_hand                                     1970 msgs (30.02 Hz):io_msgs/claws_angle[ros1msg](10) /claws_touch_data                                 1970 msgs (30.02 Hz):io_msgs/squashed_touch[ros1msg](11) /touch_visualization                              1970 msgs (30.02 Hz):sensor_msgs/CompressedImage[ros1msg](12) /mocap/sensor_data                                1970 msgs (30.02 Hz):io_msgs/squashed_mocap_data[ros1msg](13) /mocap/ros_tf                                     1970 msgs (30.02 Hz):tf2_msgs/TFMessage[ros1msg](14) /left_ee_pose                                     1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg](15) /right_ee_pose                                    1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg](16) /usb_cam_left/mjpeg_raw/compressed                1960 msgs (29.87 Hz):sensor_msgs/CompressedImage[ros1msg](17) /usb_cam_right/mjpeg_raw/compressed               1946 msgs (29.65 Hz):sensor_msgs/CompressedImage[ros1msg](18) /usb_cam_fisheye/mjpeg_raw/compressed             1957 msgs (29.82 Hz):sensor_msgs/CompressedImage[ros1msg](19) /realsense_left_hand/depth/image_rect_raw         1961 msgs (29.88 Hz):sensor_msgs/Image[ros1msg](20) /realsense_left_hand/color/image_raw/compressed   1961 msgs (29.88 Hz):sensor_msgs/CompressedImage[ros1msg](21) /realsense_right_hand/depth/image_rect_raw        1947 msgs (29.67 Hz):sensor_msgs/Image[ros1msg](22) /realsense_right_hand/color/image_raw/compressed  1947 msgs (29.67 Hz):sensor_msgs/CompressedImage[ros1msg](23) /world_left_ee_pose                               1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg](24) /world_right_ee_pose                              1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg]channels:24attachments:0metadata:0

library:mcap go v1.7.0profile:ros1messages:45200duration:1m5.625866496sstart:2025-01-15T18:09:29.628202496+08:00 (1736935769.628202496)end:2025-01-15T18:10:35.254068992+08:00 (1736935835.254068992)compression:zstd:[764/764 chunks][6.13 GiB/3.84 GiB (37.39%)][59.87 MiB/sec]channels:(1)  /rgbd/color/image_raw/compressed                  1970 msgs (30.02 Hz):sensor_msgs/CompressedImage[ros1msg](2)  /joint_states                                     1970 msgs (30.02 Hz):sensor_msgs/JointState[ros1msg](3)  /claws_r_hand                                     1970 msgs (30.02 Hz):io_msgs/claws_angle[ros1msg](4)  /global_localization                              1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg](5)  /robot_description                                   1 msgs:std_msgs/String[ros1msg](6)  /ee_visualization                                 1970 msgs (30.02 Hz):sensor_msgs/CompressedImage[ros1msg](7)  /rgbd/depth/image_raw                             1970 msgs (30.02 Hz):sensor_msgs/Image[ros1msg](8)  /colorized_depth                                  1970 msgs (30.02 Hz):sensor_msgs/CompressedImage[ros1msg](9)  /claws_l_hand                                     1970 msgs (30.02 Hz):io_msgs/claws_angle[ros1msg](10) /claws_touch_data                                 1970 msgs (30.02 Hz):io_msgs/squashed_touch[ros1msg](11) /touch_visualization                              1970 msgs (30.02 Hz):sensor_msgs/CompressedImage[ros1msg](12) /mocap/sensor_data                                1970 msgs (30.02 Hz):io_msgs/squashed_mocap_data[ros1msg](13) /mocap/ros_tf                                     1970 msgs (30.02 Hz):tf2_msgs/TFMessage[ros1msg](14) /left_ee_pose                                     1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg](15) /right_ee_pose                                    1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg](16) /usb_cam_left/mjpeg_raw/compressed                1960 msgs (29.87 Hz):sensor_msgs/CompressedImage[ros1msg](17) /usb_cam_right/mjpeg_raw/compressed               1946 msgs (29.65 Hz):sensor_msgs/CompressedImage[ros1msg](18) /usb_cam_fisheye/mjpeg_raw/compressed             1957 msgs (29.82 Hz):sensor_msgs/CompressedImage[ros1msg](19) /realsense_left_hand/depth/image_rect_raw         1961 msgs (29.88 Hz):sensor_msgs/Image[ros1msg](20) /realsense_left_hand/color/image_raw/compressed   1961 msgs (29.88 Hz):sensor_msgs/CompressedImage[ros1msg](21) /realsense_right_hand/depth/image_rect_raw        1947 msgs (29.67 Hz):sensor_msgs/Image[ros1msg](22) /realsense_right_hand/color/image_raw/compressed  1947 msgs (29.67 Hz):sensor_msgs/CompressedImage[ros1msg](23) /world_left_ee_pose                               1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg](24) /world_right_ee_pose                              1970 msgs (30.02 Hz):geometry_msgs/PoseStamped[ros1msg]channels:24attachments:0metadata:0

Topic Name

Data Meaning

/mocap/sensor_data

Joint velocity, acceleration, angular velocity, rotation angle, and sensor data from motion capture

/mocap/ros_tf

TF for all joints based on motion capture

/joint_states

JointState for all joints based on motion capture

/right_ee_pose

Right gripper pose in main head camera coordinate system

/left_ee_pose

Left gripper pose in main head camera coordinate system

/claws_l_hand

Left gripper closure degree

/claws_r_hand

Right gripper closure degree

/claws_touch_data

Gripper tactile data (contains two messages, frame_id indicates left or right gripper, first 4 values in data are valid)

/realsense_left_hand/color/image_raw/compressed

Left gripper camera RGB image

/realsense_left_hand/depth/image_rect_raw

Left gripper camera depth image

/realsense_right_hand/color/image_raw/compressed

Right gripper camera RGB image

/realsense_right_hand/depth/image_rect_raw

Right gripper camera depth image

/rgbd/color/image_raw/compressed

Main head camera RGB image

/rgbd/depth/image_raw

Main head camera depth image

/colorized_depth

Main head camera colorized depth image

/usb_cam_fisheye/mjpeg_raw/compressed

Main head fisheye camera RGB image

/usb_cam_left/mjpeg_raw/compressed

Main head left monocular camera RGB image

/usb_cam_right/mjpeg_raw/compressed

Main head right monocular camera RGB image

/ee_visualization

End-effector pose visualization in main head camera RGB image

/touch_visualization

Gripper tactile data visualization

/robot_description

Motion capture URDF

/global_localization

Main head camera pose in world coordinate system

/world_left_ee_pose

Left gripper pose in world coordinate system

/world_right_ee_pose

Right gripper pose in world coordinate system

If data is collected using tactile gloves, a tactile digital signal array topic will be added:

/mocap/touch_data 57 msgs (30.25 Hz): io_msgs/squashed_touch [ros1msg]

### Natural Language Annotationâ

{"belong_to":"20250115_InnerTest_PublicArea_TableClearing_szk_180926","mocap_offset":[],"object_set":["paper cup","placemat","trash can","napkin","plate","dinner knife","tableware storage box","wine glass","dinner fork"],"scene":"PublicArea","skill_set":["pick {A} from {B}","toss {A} into {B}","place {A} on {B}"],"subtasks":[{"skill":"pick {A} from {B}","description":"pick the paper cup from the placemat with the left gripper","description_zh":"å·¦å¤¹çª ä» é¤å« æ¡èµ· çº¸æ¯","end_frame_id":227,"end_timestamp":"1736935777206000000","sequence_id":1,"start_frame_id":159,"start_timestamp":"1736935774906000000","comment":"","attempts":"success"},{"skill":"toss {A} into {B}","description":"toss the paper cup into the trash can with the left gripper","description_zh":"å·¦å¤¹çª æçº¸æ¯è¿åå¾æ¡¶","end_frame_id":318,"end_timestamp":"1736935780244000000","sequence_id":2,"start_frame_id":231,"start_timestamp":"1736935777306000000","comment":"","attempts":"success"},...],"tag_set":[],"task_description":"20250115_InnerTest_PublicArea_TableClearing_szk_180926"}

{"belong_to":"20250115_InnerTest_PublicArea_TableClearing_szk_180926","mocap_offset":[],"object_set":["paper cup","placemat","trash can","napkin","plate","dinner knife","tableware storage box","wine glass","dinner fork"],"scene":"PublicArea","skill_set":["pick {A} from {B}","toss {A} into {B}","place {A} on {B}"],"subtasks":[{"skill":"pick {A} from {B}","description":"pick the paper cup from the placemat with the left gripper","description_zh":"å·¦å¤¹çª ä» é¤å« æ¡èµ· çº¸æ¯","end_frame_id":227,"end_timestamp":"1736935777206000000","sequence_id":1,"start_frame_id":159,"start_timestamp":"1736935774906000000","comment":"","attempts":"success"},{"skill":"toss {A} into {B}","description":"toss the paper cup into the trash can with the left gripper","description_zh":"å·¦å¤¹çª æçº¸æ¯è¿åå¾æ¡¶","end_frame_id":318,"end_timestamp":"1736935780244000000","sequence_id":2,"start_frame_id":231,"start_timestamp":"1736935777306000000","comment":"","attempts":"success"},...],"tag_set":[],"task_description":"20250115_InnerTest_PublicArea_TableClearing_szk_180926"}

## Teleoperation Robot Data Formatâ

Teleoperation robot data records the process of an operator controlling a robot through VR devices.

### Teleoperation File Structureâ

f"{robot_name}_{date}_{timestamp}_{sequence_id}"âââ RM_AIDAL_250124_172033_0.mcap    # Multimodal dataâââ RM_AIDAL_250124_172033_0.json    # Annotation dataâââ RM_AIDAL_250126_093648_0.metadata.yaml  # Metadata

f"{robot_name}_{date}_{timestamp}_{sequence_id}"âââ RM_AIDAL_250124_172033_0.mcap    # Multimodal dataâââ RM_AIDAL_250124_172033_0.json    # Annotation dataâââ RM_AIDAL_250126_093648_0.metadata.yaml  # Metadata

### Teleoperation Multimodal Dataâ

Main Topic List:

Topic Name

Data Type

Description

/camera_01/color/image_raw/compressed

/camera_01/color/image_raw/compressed

sensor_msgs/msg/CompressedImage

sensor_msgs/msg/CompressedImage

Main camera RGB image

/camera_02/color/image_raw/compressed

/camera_02/color/image_raw/compressed

sensor_msgs/msg/CompressedImage

sensor_msgs/msg/CompressedImage

Left camera RGB image

/camera_03/color/image_raw/compressed

/camera_03/color/image_raw/compressed

sensor_msgs/msg/CompressedImage

sensor_msgs/msg/CompressedImage

Right camera RGB image

io_teleop/joint_states

io_teleop/joint_states

sensor_msgs/msg/JointState

sensor_msgs/msg/JointState

Joint state

io_teleop/joint_cmd

io_teleop/joint_cmd

sensor_msgs/msg/JointState

sensor_msgs/msg/JointState

Joint command

io_teleop/target_ee_poses

io_teleop/target_ee_poses

geometry_msgs/msg/PoseArray

geometry_msgs/msg/PoseArray

Target end-effector poses

io_teleop/target_base_move

io_teleop/target_base_move

std_msgs/msg/Float64MultiArray

std_msgs/msg/Float64MultiArray

Target base move

io_teleop/target_gripper_status

io_teleop/target_gripper_status

sensor_msgs/msg/JointState

sensor_msgs/msg/JointState

Target gripper status

io_teleop/target_joint_from_vr

io_teleop/target_joint_from_vr

sensor_msgs/msg/JointState

sensor_msgs/msg/JointState

Target joints from VR device

/robot_description

/robot_description

std_msgs/msg/String

std_msgs/msg/String

Robot URDF description

/tf

/tf

tf2_msgs/msg/TFMessage

tf2_msgs/msg/TFMessage

TF spatial pose transform info

Files:RM_AIDAL_250126_091041_0.mcapBag size:443.3 MiBStorage id:mcapDuration:100.052164792sStart:Jan 24 2025 21:37:32.526605552 (1737725852.526605552)End:Jan 24 2025 21:39:12.578770344 (1737725952.578770344)Messages:62116Topic information:Topic:/camera_01/color/image_raw/compressed | Type:sensor_msgs/msg/CompressedImage | Count:3000 | Serialization Format:cdrTopic:/camera_02/color/image_raw/compressed | Type:sensor_msgs/msg/CompressedImage | Count:3000 | Serialization Format:cdrTopic:/camera_03/color/image_raw/compressed | Type:sensor_msgs/msg/CompressedImage | Count:3000 | Serialization Format:cdrTopic:io_teleop/joint_states | Type:sensor_msgs/msg/JointState | Count:1529 | Serialization Format:cdrTopic:io_teleop/joint_cmd | Type:sensor_msgs/msg/JointState | Count:10009 | Serialization Format:cdrTopic:io_teleop/target_ee_poses | Type:geometry_msgs/msg/PoseArray | Count:10014 | Serialization Format:cdrTopic:io_teleop/target_base_move | Type:std_msgs/msg/Float64MultiArray | Count:10010 | Serialization Format:cdrTopic:io_teleop/target_gripper_status | Type:sensor_msgs/msg/JointState | Count:10012 | Serialization Format:cdrTopic:io_teleop/target_joint_from_vr | Type:sensor_msgs/msg/JointState | Count:10012 | Serialization Format:cdrTopic:/robot_description | Type:std_msgs/msg/String | Count:1 | Serialization Format:cdrTopic:/tf | Type:tf2_msgs/msg/TFMessage | Count:1529 | Serialization Format:cdr

Files:RM_AIDAL_250126_091041_0.mcapBag size:443.3 MiBStorage id:mcapDuration:100.052164792sStart:Jan 24 2025 21:37:32.526605552 (1737725852.526605552)End:Jan 24 2025 21:39:12.578770344 (1737725952.578770344)Messages:62116Topic information:Topic:/camera_01/color/image_raw/compressed | Type:sensor_msgs/msg/CompressedImage | Count:3000 | Serialization Format:cdrTopic:/camera_02/color/image_raw/compressed | Type:sensor_msgs/msg/CompressedImage | Count:3000 | Serialization Format:cdrTopic:/camera_03/color/image_raw/compressed | Type:sensor_msgs/msg/CompressedImage | Count:3000 | Serialization Format:cdrTopic:io_teleop/joint_states | Type:sensor_msgs/msg/JointState | Count:1529 | Serialization Format:cdrTopic:io_teleop/joint_cmd | Type:sensor_msgs/msg/JointState | Count:10009 | Serialization Format:cdrTopic:io_teleop/target_ee_poses | Type:geometry_msgs/msg/PoseArray | Count:10014 | Serialization Format:cdrTopic:io_teleop/target_base_move | Type:std_msgs/msg/Float64MultiArray | Count:10010 | Serialization Format:cdrTopic:io_teleop/target_gripper_status | Type:sensor_msgs/msg/JointState | Count:10012 | Serialization Format:cdrTopic:io_teleop/target_joint_from_vr | Type:sensor_msgs/msg/JointState | Count:10012 | Serialization Format:cdrTopic:/robot_description | Type:std_msgs/msg/String | Count:1 | Serialization Format:cdrTopic:/tf | Type:tf2_msgs/msg/TFMessage | Count:1529 | Serialization Format:cdr

Topic Name

Data Meaning

/camera_01/color/image_raw/compressed

Main camera RGB image

/camera_02/color/image_raw/compressed

Left camera RGB image

/camera_03/color/image_raw/compressed

Right camera RGB image

io_teleop/joint_states

Joint state

io_teleop/joint_cmd

Joint command

io_teleop/target_ee_poses

Target end-effector poses

io_teleop/target_base_move

Target base move

io_teleop/target_gripper_status

Target gripper status

io_teleop/target_joint_from_vr

Target joints from VR device

/robot_description

Robot URDF description

/tf

TF spatial pose transform info

### Teleoperation Annotation Dataâ

{"belong_to":"RM_AIDAL_250126_091041_0","mocap_offset":[],"object_set":["lemon candy","plate","pistachios"],"scene":"250126","skill_set":["place {A} on {B}"],"subtasks":[{"skill":"place {A} on {B}","objecta":"lemon candy","objectb":"plate","options":["leftHand"],"description":"place the lemon candy on the plate with the left hand","end_timestamp":"1737725886915000000","sequence_id":1,"start_timestamp":"1737725880757000000","comment":"","attempts":"success"},{"skill":"place {A} on {B}","objecta":"pistachios","objectb":"plate","options":["rightHand"],"description":"place the pistachios on the plate with the right hand","end_timestamp":"1737725950745000000","sequence_id":2,"start_timestamp":"1737725941657000000","comment":"","attempts":"success"}],"tag_set":[],"task_description":"20250205_RM_ItemPacking_zhouxw"}

{"belong_to":"RM_AIDAL_250126_091041_0","mocap_offset":[],"object_set":["lemon candy","plate","pistachios"],"scene":"250126","skill_set":["place {A} on {B}"],"subtasks":[{"skill":"place {A} on {B}","objecta":"lemon candy","objectb":"plate","options":["leftHand"],"description":"place the lemon candy on the plate with the left hand","end_timestamp":"1737725886915000000","sequence_id":1,"start_timestamp":"1737725880757000000","comment":"","attempts":"success"},{"skill":"place {A} on {B}","objecta":"pistachios","objectb":"plate","options":["rightHand"],"description":"place the pistachios on the plate with the right hand","end_timestamp":"1737725950745000000","sequence_id":2,"start_timestamp":"1737725941657000000","comment":"","attempts":"success"}],"tag_set":[],"task_description":"20250205_RM_ItemPacking_zhouxw"}

## Exporting Model Training Dataâ

To facilitate model training, the platform provides various data export capabilities, converting raw captured MCAP and JSON data into formats suitable for machine learning training.

Common HDF5 and LeRobot formats can be exported with one click, and they automatically adapt to different robots or number of sensors without manual configuration.

### HDF5 Formatâ

HDF5 format is suitable for large-scale data storage and fast access, organized in a hierarchical structure.

File Structure:

chunk_001.hdf5âââ /data/                    # Data groupâ   âââ episode_001/         # First task sequenceâ   â   âââ action           # Joint commands (multi-dimensional array)â   â   âââ observation.state # Sensor observation valuesâ   â   âââ observation.gripper # Gripper stateâ   â   âââ observation.images.* # Images from various viewsâ   â   âââ task             # Task description in Englishâ   â   âââ task_zh          # Task description in Chineseâ   âââ episode_002/         # Second task sequenceâââ /meta/                   # Metadata group

chunk_001.hdf5âââ /data/                    # Data groupâ   âââ episode_001/         # First task sequenceâ   â   âââ action           # Joint commands (multi-dimensional array)â   â   âââ observation.state # Sensor observation valuesâ   â   âââ observation.gripper # Gripper stateâ   â   âââ observation.images.* # Images from various viewsâ   â   âââ task             # Task description in Englishâ   â   âââ task_zh          # Task description in Chineseâ   âââ episode_002/         # Second task sequenceâââ /meta/                   # Metadata group

Data Content:

- action- Joint control commands (float32 array)

action

- observation.state- Sensor observation values (float32 array)

observation.state

- observation.images.*- Compressed image data (JPEG format)

observation.images.*

- observation.gripper- Gripper state (float32 array)

observation.gripper

- task- English natural language description

task

- task_zh- Chinese natural language description

task_zh

- score- Action quality score

score

### LeRobot Formatâ

LeRobot format is a standard data format in the robot learning field, compatible with mainstream robot learning frameworks.

Reference Sample Data:https://huggingface.co/datasets/io-intelligence/piper_uncap_pen

Data Feature Definition:

The length and shape of the exported LeRobot dataset will automatically adapt, supporting any number of cameras or joints. The shape here is for the format exported for the AgileX desktop 7-DOF arm:

Feature Name

Data Type

Shape

Description

action

action

float32

[14]

Joint commands (7 joints per arm)

observation.state

observation.state

float32

[14]

Joint states (7 joints per arm)

observation.images.cam_high

observation.images.cam_high

image

[3,480,640]

High-view camera image

observation.images.cam_low

observation.images.cam_low

image

[3,480,640]

Low-view camera image

observation.images.cam_left_wrist

observation.images.cam_left_wrist

image

[3,480,640]

Left wrist camera image

observation.images.cam_right_wrist

observation.images.cam_right_wrist

image

[3,480,640]

Right wrist camera image

timestamp

timestamp

float32

[1]

Timestamp

frame_index

frame_index

int64

[1]

Frame index

episode_index

episode_index

int64

[1]

Task sequence index

{"codebase_version":"v2.1","robot_type":"custom_arm","total_episodes":20,"total_frames":5134,"total_tasks":20,"total_videos":0,"total_chunks":1,"chunks_size":1000,"fps":30,"splits":{"train":"0:20"},"data_path":"data/chunk-{episode_chunk:03d}/episode_{episode_index:06d}.parquet","video_path":"videos/chunk-{episode_chunk:03d}/{video_key}/episode_{episode_index:06d}.mp4","features":{"observation.images.camera_01":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_02":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_03":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_04":{"dtype":"image","shape":[480,640,3]},"observation.state":{"dtype":"float64","shape":[37],"names":["r_joint1","r_joint2","r_joint3","r_joint4","r_joint5","r_joint6","l_joint1","l_joint2","l_joint3","l_joint4","l_joint5","l_joint6","R_thumb_MCP_joint1","R_thumb_MCP_joint2","R_thumb_PIP_joint","R_thumb_DIP_joint","R_index_MCP_joint","R_index_DIP_joint","R_middle_MCP_joint","R_middle_DIP_joint","R_ring_MCP_joint","R_ring_DIP_joint","R_pinky_MCP_joint","R_pinky_DIP_joint","L_thumb_MCP_joint1","L_thumb_MCP_joint2","L_thumb_PIP_joint","L_thumb_DIP_joint","L_index_MCP_joint","L_index_DIP_joint","L_middle_MCP_joint","L_middle_DIP_joint","L_ring_MCP_joint","L_ring_DIP_joint","L_pinky_MCP_joint","L_pinky_DIP_joint","platform_joint"]},"action":{"dtype":"float64","shape":[12],"names":["l_joint1","l_joint2","l_joint3","l_joint4","l_joint5","l_joint6","r_joint1","r_joint2","r_joint3","r_joint4","r_joint5","r_joint6"]},"observation.gripper":{"dtype":"float64","shape":[2],"names":["right_gripper","left_gripper"]},"timestamp":{"dtype":"float32","shape":[1],"names":null},"frame_index":{"dtype":"int64","shape":[1],"names":null},"episode_index":{"dtype":"int64","shape":[1],"names":null},"index":{"dtype":"int64","shape":[1],"names":null},"task_index":{"dtype":"int64","shape":[1],"names":null}}}

{"codebase_version":"v2.1","robot_type":"custom_arm","total_episodes":20,"total_frames":5134,"total_tasks":20,"total_videos":0,"total_chunks":1,"chunks_size":1000,"fps":30,"splits":{"train":"0:20"},"data_path":"data/chunk-{episode_chunk:03d}/episode_{episode_index:06d}.parquet","video_path":"videos/chunk-{episode_chunk:03d}/{video_key}/episode_{episode_index:06d}.mp4","features":{"observation.images.camera_01":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_02":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_03":{"dtype":"image","shape":[480,640,3]},"observation.images.camera_04":{"dtype":"image","shape":[480,640,3]},"observation.state":{"dtype":"float64","shape":[37],"names":["r_joint1","r_joint2","r_joint3","r_joint4","r_joint5","r_joint6","l_joint1","l_joint2","l_joint3","l_joint4","l_joint5","l_joint6","R_thumb_MCP_joint1","R_thumb_MCP_joint2","R_thumb_PIP_joint","R_thumb_DIP_joint","R_index_MCP_joint","R_index_DIP_joint","R_middle_MCP_joint","R_middle_DIP_joint","R_ring_MCP_joint","R_ring_DIP_joint","R_pinky_MCP_joint","R_pinky_DIP_joint","L_thumb_MCP_joint1","L_thumb_MCP_joint2","L_thumb_PIP_joint","L_thumb_DIP_joint","L_index_MCP_joint","L_index_DIP_joint","L_middle_MCP_joint","L_middle_DIP_joint","L_ring_MCP_joint","L_ring_DIP_joint","L_pinky_MCP_joint","L_pinky_DIP_joint","platform_joint"]},"action":{"dtype":"float64","shape":[12],"names":["l_joint1","l_joint2","l_joint3","l_joint4","l_joint5","l_joint6","r_joint1","r_joint2","r_joint3","r_joint4","r_joint5","r_joint6"]},"observation.gripper":{"dtype":"float64","shape":[2],"names":["right_gripper","left_gripper"]},"timestamp":{"dtype":"float32","shape":[1],"names":null},"frame_index":{"dtype":"int64","shape":[1],"names":null},"episode_index":{"dtype":"int64","shape":[1],"names":null},"index":{"dtype":"int64","shape":[1],"names":null},"task_index":{"dtype":"int64","shape":[1],"names":null}}}