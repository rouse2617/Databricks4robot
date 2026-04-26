# HDF5æ°æ®éçæ¨¡åè®­ç»

Source: https://io-ai.tech/platform/guides/Pipeline/HDF5/training/#æ»ç»

- æ°æ®æµç¨

- HDF5æ°æ®é

- HDF5æ°æ®éçæ¨¡åè®­ç»

# HDF5æ°æ®éçæ¨¡åè®­ç»

æ¬ææ¡£ä»ç»äºå¦ä½ä½¿ç¨å¹³å°å¯¼åºçHDF5æ°æ®éè¿è¡åç§æºå¨äººå­¦ä¹ æ¨¡åçè®­ç»ï¼åæ¬æ¨¡ä»¿å­¦ä¹ ãå¼ºåå­¦ä¹ åè§è§-è¯­è¨-å¨ä½ï¼VLAï¼æ¨¡åç­ã

## æ°æ®é¢å¤çâ

å¨å¼å§è®­ç»ä¹åï¼éå¸¸éè¦å¯¹HDF5æ°æ®éè¿è¡é¢å¤çä»¥éåºä¸åçæ¨¡åæ¶æåè®­ç»æ¡æ¶ã

### åºç¡æ°æ®å è½½å¨â

importh5pyimportnumpyasnpimporttorchfromtorch.utils.dataimportDataset,DataLoaderfromPILimportImageimportioclassRobotDataset(Dataset):def__init__(self,hdf5_files,transform=None):self.hdf5_files=hdf5_filesself.transform=transformself.episodes=[]# ç´¢å¼ææepisodeforfile_pathinhdf5_files:withh5py.File(file_path,'r')asf:data_group=f['/data']forepisode_nameindata_group.keys():self.episodes.append((file_path,episode_name))def__len__(self):returnlen(self.episodes)def__getitem__(self,idx):file_path,episode_name=self.episodes[idx]withh5py.File(file_path,'r')asf:episode=f[f'/data/{episode_name}']# è¯»åå¨ä½æ°æ®actions=episode['action'][:]# è¯»åç¶ææ°æ®states=episode['observation.state'][:]# è¯»åå¤¹çªç¶ægripper=episode['observation.gripper'][:]# è¯»åå¾åæ°æ®images={}forkeyinepisode.keys():ifkey.startswith('observation.images.'):camera_name=key.split('.')[-1]# è§£åJPEGå¾åimg_data=episode[key][:]images[camera_name]=[Image.open(io.BytesIO(frame))forframeinimg_data]# è¯»åä»»å¡æè¿°task=episode.attrs.get('task','')task_zh=episode.attrs.get('task_zh','')score=episode.attrs.get('score',0.0)return{'actions':torch.FloatTensor(actions),'states':torch.FloatTensor(states),'gripper':torch.FloatTensor(gripper),'images':images,'task':task,'task_zh':task_zh,'score':score}

importh5pyimportnumpyasnpimporttorchfromtorch.utils.dataimportDataset,DataLoaderfromPILimportImageimportioclassRobotDataset(Dataset):def__init__(self,hdf5_files,transform=None):self.hdf5_files=hdf5_filesself.transform=transformself.episodes=[]# ç´¢å¼ææepisodeforfile_pathinhdf5_files:withh5py.File(file_path,'r')asf:data_group=f['/data']forepisode_nameindata_group.keys():self.episodes.append((file_path,episode_name))def__len__(self):returnlen(self.episodes)def__getitem__(self,idx):file_path,episode_name=self.episodes[idx]withh5py.File(file_path,'r')asf:episode=f[f'/data/{episode_name}']# è¯»åå¨ä½æ°æ®actions=episode['action'][:]# è¯»åç¶ææ°æ®states=episode['observation.state'][:]# è¯»åå¤¹çªç¶ægripper=episode['observation.gripper'][:]# è¯»åå¾åæ°æ®images={}forkeyinepisode.keys():ifkey.startswith('observation.images.'):camera_name=key.split('.')[-1]# è§£åJPEGå¾åimg_data=episode[key][:]images[camera_name]=[Image.open(io.BytesIO(frame))forframeinimg_data]# è¯»åä»»å¡æè¿°task=episode.attrs.get('task','')task_zh=episode.attrs.get('task_zh','')score=episode.attrs.get('score',0.0)return{'actions':torch.FloatTensor(actions),'states':torch.FloatTensor(states),'gripper':torch.FloatTensor(gripper),'images':images,'task':task,'task_zh':task_zh,'score':score}

### å¾åé¢å¤çâ

importtorchvision.transformsastransforms# å®ä¹å¾åé¢å¤çç®¡éimage_transform=transforms.Compose([transforms.Resize((224,224)),transforms.ToTensor(),transforms.Normalize(mean=[0.485,0.456,0.406],std=[0.229,0.224,0.225])])defpreprocess_images(images_dict,transform):"""é¢å¤çå¤è§è§å¾å"""processed_images={}forcamera_name,image_listinimages_dict.items():processed_images[camera_name]=torch.stack([transform(img)forimginimage_list])returnprocessed_images

importtorchvision.transformsastransforms# å®ä¹å¾åé¢å¤çç®¡éimage_transform=transforms.Compose([transforms.Resize((224,224)),transforms.ToTensor(),transforms.Normalize(mean=[0.485,0.456,0.406],std=[0.229,0.224,0.225])])defpreprocess_images(images_dict,transform):"""é¢å¤çå¤è§è§å¾å"""processed_images={}forcamera_name,image_listinimages_dict.items():processed_images[camera_name]=torch.stack([transform(img)forimginimage_list])returnprocessed_images

## æ¨¡ä»¿å­¦ä¹ ï¼Imitation Learningï¼â

æ¨¡ä»¿å­¦ä¹ éè¿å­¦ä¹ ä¸å®¶æ¼ç¤ºæ¥è®­ç»æºå¨äººç­ç¥ãHDF5æ°æ®éä¸­çé«è´¨éæ æ³¨æ°æ®éå¸¸éåè¿ç§è®­ç»æ¹å¼ã

### è¡ä¸ºåéï¼Behavior Cloningï¼â

importtorch.nnasnnimporttorch.optimasoptimclassBehaviorCloningModel(nn.Module):def__init__(self,state_dim,action_dim,image_channels=3):super().__init__()# å¾åç¼ç å¨self.image_encoder=nn.Sequential(nn.Conv2d(image_channels,32,3,stride=2,padding=1),nn.ReLU(),nn.Conv2d(32,64,3,stride=2,padding=1),nn.ReLU(),nn.Conv2d(64,128,3,stride=2,padding=1),nn.ReLU(),nn.AdaptiveAvgPool2d((4,4)),nn.Flatten(),nn.Linear(128*4*4,256))# ç¶æç¼ç å¨self.state_encoder=nn.Sequential(nn.Linear(state_dim,128),nn.ReLU(),nn.Linear(128,128))# èåå±self.fusion=nn.Sequential(nn.Linear(256+128,256),nn.ReLU(),nn.Linear(256,128),nn.ReLU(),nn.Linear(128,action_dim))defforward(self,images,states):# å¤çå¤è§è§å¾åï¼è¿éç®åä¸ºä½¿ç¨ç¬¬ä¸ä¸ªç¸æºï¼img_features=self.image_encoder(images)state_features=self.state_encoder(states)# ç¹å¾èåcombined=torch.cat([img_features,state_features],dim=1)actions=self.fusion(combined)returnactions# è®­ç»å¾ªç¯deftrain_bc_model(model,dataloader,num_epochs=100):criterion=nn.MSELoss()optimizer=optim.Adam(model.parameters(),lr=1e-3)forepochinrange(num_epochs):total_loss=0forbatchindataloader:# è·åç¬¬ä¸ä¸ªç¸æºçå¾åï¼ç®åå¤çï¼images=list(batch['images'].values())[0][:,0]# [batch, H, W, C]images=images.permute(0,3,1,2)# [batch, C, H, W]states=batch['states'][:,0]# ç¬¬ä¸ä¸ªæ¶é´æ­¥çç¶æactions=batch['actions'][:,0]# ç¬¬ä¸ä¸ªæ¶é´æ­¥çå¨ä½optimizer.zero_grad()predicted_actions=model(images,states)loss=criterion(predicted_actions,actions)loss.backward()optimizer.step()total_loss+=loss.item()print(f'Epoch{epoch+1}/{num_epochs}, Loss:{total_loss/len(dataloader):.4f}')

importtorch.nnasnnimporttorch.optimasoptimclassBehaviorCloningModel(nn.Module):def__init__(self,state_dim,action_dim,image_channels=3):super().__init__()# å¾åç¼ç å¨self.image_encoder=nn.Sequential(nn.Conv2d(image_channels,32,3,stride=2,padding=1),nn.ReLU(),nn.Conv2d(32,64,3,stride=2,padding=1),nn.ReLU(),nn.Conv2d(64,128,3,stride=2,padding=1),nn.ReLU(),nn.AdaptiveAvgPool2d((4,4)),nn.Flatten(),nn.Linear(128*4*4,256))# ç¶æç¼ç å¨self.state_encoder=nn.Sequential(nn.Linear(state_dim,128),nn.ReLU(),nn.Linear(128,128))# èåå±self.fusion=nn.Sequential(nn.Linear(256+128,256),nn.ReLU(),nn.Linear(256,128),nn.ReLU(),nn.Linear(128,action_dim))defforward(self,images,states):# å¤çå¤è§è§å¾åï¼è¿éç®åä¸ºä½¿ç¨ç¬¬ä¸ä¸ªç¸æºï¼img_features=self.image_encoder(images)state_features=self.state_encoder(states)# ç¹å¾èåcombined=torch.cat([img_features,state_features],dim=1)actions=self.fusion(combined)returnactions# è®­ç»å¾ªç¯deftrain_bc_model(model,dataloader,num_epochs=100):criterion=nn.MSELoss()optimizer=optim.Adam(model.parameters(),lr=1e-3)forepochinrange(num_epochs):total_loss=0forbatchindataloader:# è·åç¬¬ä¸ä¸ªç¸æºçå¾åï¼ç®åå¤çï¼images=list(batch['images'].values())[0][:,0]# [batch, H, W, C]images=images.permute(0,3,1,2)# [batch, C, H, W]states=batch['states'][:,0]# ç¬¬ä¸ä¸ªæ¶é´æ­¥çç¶æactions=batch['actions'][:,0]# ç¬¬ä¸ä¸ªæ¶é´æ­¥çå¨ä½optimizer.zero_grad()predicted_actions=model(images,states)loss=criterion(predicted_actions,actions)loss.backward()optimizer.step()total_loss+=loss.item()print(f'Epoch{epoch+1}/{num_epochs}, Loss:{total_loss/len(dataloader):.4f}')

## åºåå»ºæ¨¡ä¸Transformerâ

å¯¹äºéè¦èèæ¶åºä¿¡æ¯çä»»å¡ï¼å¯ä»¥ä½¿ç¨Transformeræ¶ææ¥å»ºæ¨¡å¨ä½åºåã

### Transformerç­ç¥ç½ç»â

classTransformerPolicy(nn.Module):def__init__(self,state_dim,action_dim,seq_len=50,d_model=256):super().__init__()self.seq_len=seq_lenself.d_model=d_model# è¾å¥æå½±self.state_proj=nn.Linear(state_dim,d_model)self.action_proj=nn.Linear(action_dim,d_model)# ä½ç½®ç¼ç self.pos_encoding=nn.Parameter(torch.randn(seq_len,d_model))# Transformerç¼ç å¨encoder_layer=nn.TransformerEncoderLayer(d_model=d_model,nhead=8,batch_first=True)self.transformer=nn.TransformerEncoder(encoder_layer,num_layers=6)# è¾åºå±self.output_proj=nn.Linear(d_model,action_dim)defforward(self,states,actions=None):batch_size,seq_len=states.shape[:2]# ç¶æç¼ç state_emb=self.state_proj(states)ifactionsisnotNone:# è®­ç»æ¨¡å¼action_emb=self.action_proj(actions)# å°å¨ä½åå³ç§»ä½ä½ä¸ºè¾å¥action_input=torch.cat([torch.zeros(batch_size,1,self.d_model,device=actions.device),action_emb[:,:-1]],dim=1)inputs=state_emb+action_inputelse:# æ¨çæ¨¡å¼inputs=state_emb# æ·»å ä½ç½®ç¼ç inputs+=self.pos_encoding[:seq_len]# Transformerå¤çoutputs=self.transformer(inputs)# é¢æµå¨ä½predicted_actions=self.output_proj(outputs)returnpredicted_actions

classTransformerPolicy(nn.Module):def__init__(self,state_dim,action_dim,seq_len=50,d_model=256):super().__init__()self.seq_len=seq_lenself.d_model=d_model# è¾å¥æå½±self.state_proj=nn.Linear(state_dim,d_model)self.action_proj=nn.Linear(action_dim,d_model)# ä½ç½®ç¼ç self.pos_encoding=nn.Parameter(torch.randn(seq_len,d_model))# Transformerç¼ç å¨encoder_layer=nn.TransformerEncoderLayer(d_model=d_model,nhead=8,batch_first=True)self.transformer=nn.TransformerEncoder(encoder_layer,num_layers=6)# è¾åºå±self.output_proj=nn.Linear(d_model,action_dim)defforward(self,states,actions=None):batch_size,seq_len=states.shape[:2]# ç¶æç¼ç state_emb=self.state_proj(states)ifactionsisnotNone:# è®­ç»æ¨¡å¼action_emb=self.action_proj(actions)# å°å¨ä½åå³ç§»ä½ä½ä¸ºè¾å¥action_input=torch.cat([torch.zeros(batch_size,1,self.d_model,device=actions.device),action_emb[:,:-1]],dim=1)inputs=state_emb+action_inputelse:# æ¨çæ¨¡å¼inputs=state_emb# æ·»å ä½ç½®ç¼ç inputs+=self.pos_encoding[:seq_len]# Transformerå¤çoutputs=self.transformer(inputs)# é¢æµå¨ä½predicted_actions=self.output_proj(outputs)returnpredicted_actions

## è§è§-è¯­è¨-å¨ä½ï¼VLAï¼æ¨¡åâ

VLAæ¨¡åç»åè§è§ãè¯­è¨åå¨ä½ä¿¡æ¯ï¼è½å¤æ ¹æ®èªç¶è¯­è¨æä»¤æ§è¡å¤æçæºå¨äººä»»å¡ã

### å¤æ¨¡æVLAæ¶æâ

fromtransformersimportAutoTokenizer,AutoModelclassVLAModel(nn.Module):def__init__(self,action_dim,language_model='bert-base-uncased'):super().__init__()# è¯­è¨ç¼ç å¨self.tokenizer=AutoTokenizer.from_pretrained(language_model)self.language_encoder=AutoModel.from_pretrained(language_model)# è§è§ç¼ç å¨self.vision_encoder=nn.Sequential(nn.Conv2d(3,64,7,stride=2,padding=3),nn.ReLU(),nn.MaxPool2d(2),nn.Conv2d(64,128,3,padding=1),nn.ReLU(),nn.Conv2d(128,256,3,padding=1),nn.ReLU(),nn.AdaptiveAvgPool2d((8,8)),nn.Flatten(),nn.Linear(256*8*8,512))# è·¨æ¨¡ææ³¨æåself.cross_attention=nn.MultiheadAttention(embed_dim=512,num_heads=8,batch_first=True)# å¨ä½è§£ç å¨self.action_decoder=nn.Sequential(nn.Linear(512,256),nn.ReLU(),nn.Linear(256,128),nn.ReLU(),nn.Linear(128,action_dim))defforward(self,images,task_descriptions,states):batch_size=images.shape[0]# ç¼ç è¯­è¨æä»¤tokens=self.tokenizer(task_descriptions,return_tensors='pt',padding=True,truncation=True,max_length=128)language_features=self.language_encoder(**tokens).last_hidden_state# ç¼ç è§è§ä¿¡æ¯vision_features=self.vision_encoder(images)vision_features=vision_features.unsqueeze(1)# [batch, 1, 512]# è·¨æ¨¡ææ³¨æåattended_features,_=self.cross_attention(vision_features,language_features,language_features)# é¢æµå¨ä½actions=self.action_decoder(attended_features.squeeze(1))returnactions# VLAè®­ç»å½æ°deftrain_vla_model(model,dataloader,num_epochs=50):criterion=nn.MSELoss()optimizer=optim.Adam(model.parameters(),lr=1e-4)forepochinrange(num_epochs):total_loss=0forbatchindataloader:# è·åæ°æ®images=list(batch['images'].values())[0][:,0].permute(0,3,1,2)task_descriptions=batch['task']actions=batch['actions'][:,0]states=batch['states'][:,0]optimizer.zero_grad()predicted_actions=model(images,task_descriptions,states)loss=criterion(predicted_actions,actions)loss.backward()optimizer.step()total_loss+=loss.item()print(f'Epoch{epoch+1}/{num_epochs}, Loss:{total_loss/len(dataloader):.4f}')

fromtransformersimportAutoTokenizer,AutoModelclassVLAModel(nn.Module):def__init__(self,action_dim,language_model='bert-base-uncased'):super().__init__()# è¯­è¨ç¼ç å¨self.tokenizer=AutoTokenizer.from_pretrained(language_model)self.language_encoder=AutoModel.from_pretrained(language_model)# è§è§ç¼ç å¨self.vision_encoder=nn.Sequential(nn.Conv2d(3,64,7,stride=2,padding=3),nn.ReLU(),nn.MaxPool2d(2),nn.Conv2d(64,128,3,padding=1),nn.ReLU(),nn.Conv2d(128,256,3,padding=1),nn.ReLU(),nn.AdaptiveAvgPool2d((8,8)),nn.Flatten(),nn.Linear(256*8*8,512))# è·¨æ¨¡ææ³¨æåself.cross_attention=nn.MultiheadAttention(embed_dim=512,num_heads=8,batch_first=True)# å¨ä½è§£ç å¨self.action_decoder=nn.Sequential(nn.Linear(512,256),nn.ReLU(),nn.Linear(256,128),nn.ReLU(),nn.Linear(128,action_dim))defforward(self,images,task_descriptions,states):batch_size=images.shape[0]# ç¼ç è¯­è¨æä»¤tokens=self.tokenizer(task_descriptions,return_tensors='pt',padding=True,truncation=True,max_length=128)language_features=self.language_encoder(**tokens).last_hidden_state# ç¼ç è§è§ä¿¡æ¯vision_features=self.vision_encoder(images)vision_features=vision_features.unsqueeze(1)# [batch, 1, 512]# è·¨æ¨¡ææ³¨æåattended_features,_=self.cross_attention(vision_features,language_features,language_features)# é¢æµå¨ä½actions=self.action_decoder(attended_features.squeeze(1))returnactions# VLAè®­ç»å½æ°deftrain_vla_model(model,dataloader,num_epochs=50):criterion=nn.MSELoss()optimizer=optim.Adam(model.parameters(),lr=1e-4)forepochinrange(num_epochs):total_loss=0forbatchindataloader:# è·åæ°æ®images=list(batch['images'].values())[0][:,0].permute(0,3,1,2)task_descriptions=batch['task']actions=batch['actions'][:,0]states=batch['states'][:,0]optimizer.zero_grad()predicted_actions=model(images,task_descriptions,states)loss=criterion(predicted_actions,actions)loss.backward()optimizer.step()total_loss+=loss.item()print(f'Epoch{epoch+1}/{num_epochs}, Loss:{total_loss/len(dataloader):.4f}')

## å¼ºåå­¦ä¹ éæâ

å¯ä»¥å°HDF5æ°æ®ä½ä¸ºå¼ºåå­¦ä¹ çåå§åæ°æ®æç»éªåæ¾ç¼å²åºçç§å­æ°æ®ã

### ç¦»çº¿å¼ºåå­¦ä¹â

importtorch.nn.functionalasFclassOfflineRLAgent:def__init__(self,state_dim,action_dim,lr=3e-4):self.actor=BehaviorCloningModel(state_dim,action_dim)self.critic=nn.Sequential(nn.Linear(state_dim+action_dim,256),nn.ReLU(),nn.Linear(256,256),nn.ReLU(),nn.Linear(256,1))self.actor_optimizer=optim.Adam(self.actor.parameters(),lr=lr)self.critic_optimizer=optim.Adam(self.critic.parameters(),lr=lr)deftrain_step(self,states,actions,rewards,next_states,dones):# è®­ç»Criticwithtorch.no_grad():next_actions=self.actor(next_states)target_q=rewards+0.99*(1-dones)*self.critic(torch.cat([next_states,next_actions],dim=1))current_q=self.critic(torch.cat([states,actions],dim=1))critic_loss=F.mse_loss(current_q,target_q)self.critic_optimizer.zero_grad()critic_loss.backward()self.critic_optimizer.step()# è®­ç»Actorpredicted_actions=self.actor(states)actor_loss=-self.critic(torch.cat([states,predicted_actions],dim=1)).mean()self.actor_optimizer.zero_grad()actor_loss.backward()self.actor_optimizer.step()returncritic_loss.item(),actor_loss.item()

importtorch.nn.functionalasFclassOfflineRLAgent:def__init__(self,state_dim,action_dim,lr=3e-4):self.actor=BehaviorCloningModel(state_dim,action_dim)self.critic=nn.Sequential(nn.Linear(state_dim+action_dim,256),nn.ReLU(),nn.Linear(256,256),nn.ReLU(),nn.Linear(256,1))self.actor_optimizer=optim.Adam(self.actor.parameters(),lr=lr)self.critic_optimizer=optim.Adam(self.critic.parameters(),lr=lr)deftrain_step(self,states,actions,rewards,next_states,dones):# è®­ç»Criticwithtorch.no_grad():next_actions=self.actor(next_states)target_q=rewards+0.99*(1-dones)*self.critic(torch.cat([next_states,next_actions],dim=1))current_q=self.critic(torch.cat([states,actions],dim=1))critic_loss=F.mse_loss(current_q,target_q)self.critic_optimizer.zero_grad()critic_loss.backward()self.critic_optimizer.step()# è®­ç»Actorpredicted_actions=self.actor(states)actor_loss=-self.critic(torch.cat([states,predicted_actions],dim=1)).mean()self.actor_optimizer.zero_grad()actor_loss.backward()self.actor_optimizer.step()returncritic_loss.item(),actor_loss.item()

## æ°æ®å¢å¼ºææ¯â

ä¸ºäºæé«æ¨¡åçæ³åè½åï¼å¯ä»¥å¯¹HDF5æ°æ®è¿è¡åç§å¢å¼ºã

### å¾åå¢å¼ºâ

importtorchvision.transformsasTclassRobotDataAugmentation:def__init__(self):self.image_aug=T.Compose([T.ColorJitter(brightness=0.2,contrast=0.2,saturation=0.2,hue=0.1),T.RandomRotation(degrees=5),T.RandomResizedCrop(224,scale=(0.9,1.0)),T.RandomHorizontalFlip(p=0.1),# å°æ¦çæ°´å¹³ç¿»è½¬])defaugment_episode(self,episode_data):"""å¯¹åä¸ªepisodeè¿è¡æ°æ®å¢å¼º"""augmented_data=episode_data.copy()# å¾åå¢å¼ºforcamera_name,imagesinepisode_data['images'].items():augmented_images=[]forimginimages:iftorch.rand(1)<0.5:# 50%æ¦çè¿è¡å¢å¼ºimg=self.image_aug(img)augmented_images.append(img)augmented_data['images'][camera_name]=augmented_images# å¨ä½åªå£°iftorch.rand(1)<0.3:# 30%æ¦çæ·»å å¨ä½åªå£°noise=torch.randn_like(episode_data['actions'])*0.01augmented_data['actions']=episode_data['actions']+noisereturnaugmented_data

importtorchvision.transformsasTclassRobotDataAugmentation:def__init__(self):self.image_aug=T.Compose([T.ColorJitter(brightness=0.2,contrast=0.2,saturation=0.2,hue=0.1),T.RandomRotation(degrees=5),T.RandomResizedCrop(224,scale=(0.9,1.0)),T.RandomHorizontalFlip(p=0.1),# å°æ¦çæ°´å¹³ç¿»è½¬])defaugment_episode(self,episode_data):"""å¯¹åä¸ªepisodeè¿è¡æ°æ®å¢å¼º"""augmented_data=episode_data.copy()# å¾åå¢å¼ºforcamera_name,imagesinepisode_data['images'].items():augmented_images=[]forimginimages:iftorch.rand(1)<0.5:# 50%æ¦çè¿è¡å¢å¼ºimg=self.image_aug(img)augmented_images.append(img)augmented_data['images'][camera_name]=augmented_images# å¨ä½åªå£°iftorch.rand(1)<0.3:# 30%æ¦çæ·»å å¨ä½åªå£°noise=torch.randn_like(episode_data['actions'])*0.01augmented_data['actions']=episode_data['actions']+noisereturnaugmented_data

## æ¨¡åè¯ä¼°ä¸é¨ç½²â

### è¯ä¼°ææ â

defevaluate_model(model,test_dataloader,device):model.eval()total_mse=0total_samples=0withtorch.no_grad():forbatchintest_dataloader:images=list(batch['images'].values())[0][:,0].permute(0,3,1,2).to(device)states=batch['states'][:,0].to(device)true_actions=batch['actions'][:,0].to(device)predicted_actions=model(images,states)mse=F.mse_loss(predicted_actions,true_actions)total_mse+=mse.item()*len(true_actions)total_samples+=len(true_actions)avg_mse=total_mse/total_samplesprint(f'Test MSE:{avg_mse:.6f}')returnavg_mse

defevaluate_model(model,test_dataloader,device):model.eval()total_mse=0total_samples=0withtorch.no_grad():forbatchintest_dataloader:images=list(batch['images'].values())[0][:,0].permute(0,3,1,2).to(device)states=batch['states'][:,0].to(device)true_actions=batch['actions'][:,0].to(device)predicted_actions=model(images,states)mse=F.mse_loss(predicted_actions,true_actions)total_mse+=mse.item()*len(true_actions)total_samples+=len(true_actions)avg_mse=total_mse/total_samplesprint(f'Test MSE:{avg_mse:.6f}')returnavg_mse

### æ¨¡åä¿å­ä¸å è½½â

defsave_model(model,optimizer,epoch,loss,filepath):torch.save({'epoch':epoch,'model_state_dict':model.state_dict(),'optimizer_state_dict':optimizer.state_dict(),'loss':loss,},filepath)defload_model(model,optimizer,filepath):checkpoint=torch.load(filepath)model.load_state_dict(checkpoint['model_state_dict'])optimizer.load_state_dict(checkpoint['optimizer_state_dict'])epoch=checkpoint['epoch']loss=checkpoint['loss']returnepoch,loss

defsave_model(model,optimizer,epoch,loss,filepath):torch.save({'epoch':epoch,'model_state_dict':model.state_dict(),'optimizer_state_dict':optimizer.state_dict(),'loss':loss,},filepath)defload_model(model,optimizer,filepath):checkpoint=torch.load(filepath)model.load_state_dict(checkpoint['model_state_dict'])optimizer.load_state_dict(checkpoint['optimizer_state_dict'])epoch=checkpoint['epoch']loss=checkpoint['loss']returnepoch,loss

## æ»ç»â

éè¿åçå©ç¨å¹³å°å¯¼åºçHDF5æ°æ®ï¼å¯ä»¥ææè®­ç»åç§æºå¨äººå­¦ä¹ æ¨¡åï¼

- æ¨¡ä»¿å­¦ä¹ï¼ç´æ¥ä»ä¸å®¶æ¼ç¤ºå­¦ä¹ ç­ç¥

- åºåå»ºæ¨¡ï¼ä½¿ç¨Transformerå¤çæ¶åºä¾èµ

- å¤æ¨¡æå­¦ä¹ï¼ç»åè§è§ãè¯­è¨åå¨ä½ä¿¡æ¯

- å¼ºåå­¦ä¹ï¼ä½ä¸ºç¦»çº¿æ°æ®æåå§åæ°æ®

å³é®æ¯æ ¹æ®å·ä½ä»»å¡éæ±éæ©åéçæ¨¡åæ¶æï¼å¹¶ååå©ç¨HDF5æ°æ®çä¸°å¯æ æ³¨ä¿¡æ¯æ¥æé«æ¨¡åæ§è½ã