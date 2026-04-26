# Training with HDF5

Source: https://io-ai.tech/platform/en/guides/Pipeline/HDF5/training/#summary

- Data Pipeline

- HDF5 Dataset

- Training with HDF5

# Training with HDF5

This page explains how to train robot learning models using exported HDF5 datasets: imitation learning, RL, and VLA.

## Preprocessingâ

Prepare data to match your model/framework.

### Basic loaderâ

importh5pyimportnumpyasnpimporttorchfromtorch.utils.dataimportDataset,DataLoaderfromPILimportImageimportioclassRobotDataset(Dataset):def__init__(self,hdf5_files,transform=None):self.hdf5_files=hdf5_filesself.transform=transformself.episodes=[]forfile_pathinhdf5_files:withh5py.File(file_path,'r')asf:data_group=f['/data']forepisode_nameindata_group.keys():self.episodes.append((file_path,episode_name))def__len__(self):returnlen(self.episodes)def__getitem__(self,idx):file_path,episode_name=self.episodes[idx]withh5py.File(file_path,'r')asf:episode=f[f'/data/{episode_name}']actions=episode['action'][:]states=episode['observation.state'][:]gripper=episode['observation.gripper'][:]images={}forkeyinepisode.keys():ifkey.startswith('observation.images.'):cam=key.split('.')[-1]img_data=episode[key][:]images[cam]=[Image.open(io.BytesIO(frame))forframeinimg_data]task=episode.attrs.get('task','')task_zh=episode.attrs.get('task_zh','')score=episode.attrs.get('score',0.0)return{'actions':torch.FloatTensor(actions),'states':torch.FloatTensor(states),'gripper':torch.FloatTensor(gripper),'images':images,'task':task,'task_zh':task_zh,'score':score,}

importh5pyimportnumpyasnpimporttorchfromtorch.utils.dataimportDataset,DataLoaderfromPILimportImageimportioclassRobotDataset(Dataset):def__init__(self,hdf5_files,transform=None):self.hdf5_files=hdf5_filesself.transform=transformself.episodes=[]forfile_pathinhdf5_files:withh5py.File(file_path,'r')asf:data_group=f['/data']forepisode_nameindata_group.keys():self.episodes.append((file_path,episode_name))def__len__(self):returnlen(self.episodes)def__getitem__(self,idx):file_path,episode_name=self.episodes[idx]withh5py.File(file_path,'r')asf:episode=f[f'/data/{episode_name}']actions=episode['action'][:]states=episode['observation.state'][:]gripper=episode['observation.gripper'][:]images={}forkeyinepisode.keys():ifkey.startswith('observation.images.'):cam=key.split('.')[-1]img_data=episode[key][:]images[cam]=[Image.open(io.BytesIO(frame))forframeinimg_data]task=episode.attrs.get('task','')task_zh=episode.attrs.get('task_zh','')score=episode.attrs.get('score',0.0)return{'actions':torch.FloatTensor(actions),'states':torch.FloatTensor(states),'gripper':torch.FloatTensor(gripper),'images':images,'task':task,'task_zh':task_zh,'score':score,}

### Image transformsâ

importtorchvision.transformsastransformsimage_transform=transforms.Compose([transforms.Resize((224,224)),transforms.ToTensor(),transforms.Normalize(mean=[0.485,0.456,0.406],std=[0.229,0.224,0.225]),])defpreprocess_images(images_dict,transform):processed={}forcam,imgsinimages_dict.items():processed[cam]=torch.stack([transform(img)forimginimgs])returnprocessed

importtorchvision.transformsastransformsimage_transform=transforms.Compose([transforms.Resize((224,224)),transforms.ToTensor(),transforms.Normalize(mean=[0.485,0.456,0.406],std=[0.229,0.224,0.225]),])defpreprocess_images(images_dict,transform):processed={}forcam,imgsinimages_dict.items():processed[cam]=torch.stack([transform(img)forimginimgs])returnprocessed

## Imitation learningâ

### Behavior cloningâ

importtorch.nnasnnimporttorch.optimasoptimclassBehaviorCloningModel(nn.Module):def__init__(self,state_dim,action_dim,image_channels=3):super().__init__()self.image_encoder=nn.Sequential(nn.Conv2d(image_channels,32,3,stride=2,padding=1),nn.ReLU(),nn.Conv2d(32,64,3,stride=2,padding=1),nn.ReLU(),nn.Conv2d(64,128,3,stride=2,padding=1),nn.ReLU(),nn.AdaptiveAvgPool2d((4,4)),nn.Flatten(),nn.Linear(128*4*4,256))self.state_encoder=nn.Sequential(nn.Linear(state_dim,128),nn.ReLU(),nn.Linear(128,128))self.fusion=nn.Sequential(nn.Linear(256+128,256),nn.ReLU(),nn.Linear(256,128),nn.ReLU(),nn.Linear(128,action_dim))defforward(self,images,states):img_feat=self.image_encoder(images)st_feat=self.state_encoder(states)returnself.fusion(torch.cat([img_feat,st_feat],dim=1))deftrain_bc_model(model,dataloader,num_epochs=100):criterion=nn.MSELoss()optimizer=optim.Adam(model.parameters(),lr=1e-3)forepochinrange(num_epochs):total=0forbatchindataloader:images=list(batch['images'].values())[0][:,0].permute(0,3,1,2)states=batch['states'][:,0]actions=batch['actions'][:,0]optimizer.zero_grad()loss=criterion(model(images,states),actions)loss.backward();optimizer.step()total+=loss.item()print(f'Epoch{epoch+1}/{num_epochs}, Loss:{total/len(dataloader):.4f}')

importtorch.nnasnnimporttorch.optimasoptimclassBehaviorCloningModel(nn.Module):def__init__(self,state_dim,action_dim,image_channels=3):super().__init__()self.image_encoder=nn.Sequential(nn.Conv2d(image_channels,32,3,stride=2,padding=1),nn.ReLU(),nn.Conv2d(32,64,3,stride=2,padding=1),nn.ReLU(),nn.Conv2d(64,128,3,stride=2,padding=1),nn.ReLU(),nn.AdaptiveAvgPool2d((4,4)),nn.Flatten(),nn.Linear(128*4*4,256))self.state_encoder=nn.Sequential(nn.Linear(state_dim,128),nn.ReLU(),nn.Linear(128,128))self.fusion=nn.Sequential(nn.Linear(256+128,256),nn.ReLU(),nn.Linear(256,128),nn.ReLU(),nn.Linear(128,action_dim))defforward(self,images,states):img_feat=self.image_encoder(images)st_feat=self.state_encoder(states)returnself.fusion(torch.cat([img_feat,st_feat],dim=1))deftrain_bc_model(model,dataloader,num_epochs=100):criterion=nn.MSELoss()optimizer=optim.Adam(model.parameters(),lr=1e-3)forepochinrange(num_epochs):total=0forbatchindataloader:images=list(batch['images'].values())[0][:,0].permute(0,3,1,2)states=batch['states'][:,0]actions=batch['actions'][:,0]optimizer.zero_grad()loss=criterion(model(images,states),actions)loss.backward();optimizer.step()total+=loss.item()print(f'Epoch{epoch+1}/{num_epochs}, Loss:{total/len(dataloader):.4f}')

## Sequence modeling (Transformer)â

classTransformerPolicy(nn.Module):def__init__(self,state_dim,action_dim,seq_len=50,d_model=256):super().__init__()self.seq_len,self.d_model=seq_len,d_modelself.state_proj=nn.Linear(state_dim,d_model)self.action_proj=nn.Linear(action_dim,d_model)self.pos_encoding=nn.Parameter(torch.randn(seq_len,d_model))enc_layer=nn.TransformerEncoderLayer(d_model=d_model,nhead=8,batch_first=True)self.transformer=nn.TransformerEncoder(enc_layer,num_layers=6)self.output_proj=nn.Linear(d_model,action_dim)defforward(self,states,actions=None):bsz,seq_len=states.shape[:2]st=self.state_proj(states)ifactionsisnotNone:act=self.action_proj(actions)act_in=torch.cat([torch.zeros(bsz,1,self.d_model,device=actions.device),act[:,:-1]],dim=1)inputs=st+act_inelse:inputs=stinputs+=self.pos_encoding[:seq_len]out=self.transformer(inputs)returnself.output_proj(out)

classTransformerPolicy(nn.Module):def__init__(self,state_dim,action_dim,seq_len=50,d_model=256):super().__init__()self.seq_len,self.d_model=seq_len,d_modelself.state_proj=nn.Linear(state_dim,d_model)self.action_proj=nn.Linear(action_dim,d_model)self.pos_encoding=nn.Parameter(torch.randn(seq_len,d_model))enc_layer=nn.TransformerEncoderLayer(d_model=d_model,nhead=8,batch_first=True)self.transformer=nn.TransformerEncoder(enc_layer,num_layers=6)self.output_proj=nn.Linear(d_model,action_dim)defforward(self,states,actions=None):bsz,seq_len=states.shape[:2]st=self.state_proj(states)ifactionsisnotNone:act=self.action_proj(actions)act_in=torch.cat([torch.zeros(bsz,1,self.d_model,device=actions.device),act[:,:-1]],dim=1)inputs=st+act_inelse:inputs=stinputs+=self.pos_encoding[:seq_len]out=self.transformer(inputs)returnself.output_proj(out)

## VLA model (visionâlanguageâaction)â

fromtransformersimportAutoTokenizer,AutoModelclassVLAModel(nn.Module):def__init__(self,action_dim,language_model='bert-base-uncased'):super().__init__()self.tokenizer=AutoTokenizer.from_pretrained(language_model)self.language_encoder=AutoModel.from_pretrained(language_model)self.vision_encoder=nn.Sequential(nn.Conv2d(3,64,7,stride=2,padding=3),nn.ReLU(),nn.MaxPool2d(2),nn.Conv2d(64,128,3,padding=1),nn.ReLU(),nn.Conv2d(128,256,3,padding=1),nn.ReLU(),nn.AdaptiveAvgPool2d((8,8)),nn.Flatten(),nn.Linear(256*8*8,512))self.cross_attention=nn.MultiheadAttention(embed_dim=512,num_heads=8,batch_first=True)self.action_decoder=nn.Sequential(nn.Linear(512,256),nn.ReLU(),nn.Linear(256,128),nn.ReLU(),nn.Linear(128,action_dim))defforward(self,images,task_descriptions,states):tokens=self.tokenizer(task_descriptions,return_tensors='pt',padding=True,truncation=True,max_length=128)lang=self.language_encoder(**tokens).last_hidden_statevis=self.vision_encoder(images).unsqueeze(1)attn,_=self.cross_attention(vis,lang,lang)returnself.action_decoder(attn.squeeze(1))

fromtransformersimportAutoTokenizer,AutoModelclassVLAModel(nn.Module):def__init__(self,action_dim,language_model='bert-base-uncased'):super().__init__()self.tokenizer=AutoTokenizer.from_pretrained(language_model)self.language_encoder=AutoModel.from_pretrained(language_model)self.vision_encoder=nn.Sequential(nn.Conv2d(3,64,7,stride=2,padding=3),nn.ReLU(),nn.MaxPool2d(2),nn.Conv2d(64,128,3,padding=1),nn.ReLU(),nn.Conv2d(128,256,3,padding=1),nn.ReLU(),nn.AdaptiveAvgPool2d((8,8)),nn.Flatten(),nn.Linear(256*8*8,512))self.cross_attention=nn.MultiheadAttention(embed_dim=512,num_heads=8,batch_first=True)self.action_decoder=nn.Sequential(nn.Linear(512,256),nn.ReLU(),nn.Linear(256,128),nn.ReLU(),nn.Linear(128,action_dim))defforward(self,images,task_descriptions,states):tokens=self.tokenizer(task_descriptions,return_tensors='pt',padding=True,truncation=True,max_length=128)lang=self.language_encoder(**tokens).last_hidden_statevis=self.vision_encoder(images).unsqueeze(1)attn,_=self.cross_attention(vis,lang,lang)returnself.action_decoder(attn.squeeze(1))

## Offline RL exampleâ

importtorch.nn.functionalasFclassOfflineRLAgent:def__init__(self,state_dim,action_dim,lr=3e-4):self.actor=BehaviorCloningModel(state_dim,action_dim)self.critic=nn.Sequential(nn.Linear(state_dim+action_dim,256),nn.ReLU(),nn.Linear(256,256),nn.ReLU(),nn.Linear(256,1))self.actor_optimizer=optim.Adam(self.actor.parameters(),lr=lr)self.critic_optimizer=optim.Adam(self.critic.parameters(),lr=lr)deftrain_step(self,states,actions,rewards,next_states,dones):withtorch.no_grad():next_actions=self.actor(next_states)target_q=rewards+0.99*(1-dones)*self.critic(torch.cat([next_states,next_actions],dim=1))current_q=self.critic(torch.cat([states,actions],dim=1))critic_loss=F.mse_loss(current_q,target_q)self.critic_optimizer.zero_grad();critic_loss.backward();self.critic_optimizer.step()pred=self.actor(states)actor_loss=-self.critic(torch.cat([states,pred],dim=1)).mean()self.actor_optimizer.zero_grad();actor_loss.backward();self.actor_optimizer.step()returncritic_loss.item(),actor_loss.item()

importtorch.nn.functionalasFclassOfflineRLAgent:def__init__(self,state_dim,action_dim,lr=3e-4):self.actor=BehaviorCloningModel(state_dim,action_dim)self.critic=nn.Sequential(nn.Linear(state_dim+action_dim,256),nn.ReLU(),nn.Linear(256,256),nn.ReLU(),nn.Linear(256,1))self.actor_optimizer=optim.Adam(self.actor.parameters(),lr=lr)self.critic_optimizer=optim.Adam(self.critic.parameters(),lr=lr)deftrain_step(self,states,actions,rewards,next_states,dones):withtorch.no_grad():next_actions=self.actor(next_states)target_q=rewards+0.99*(1-dones)*self.critic(torch.cat([next_states,next_actions],dim=1))current_q=self.critic(torch.cat([states,actions],dim=1))critic_loss=F.mse_loss(current_q,target_q)self.critic_optimizer.zero_grad();critic_loss.backward();self.critic_optimizer.step()pred=self.actor(states)actor_loss=-self.critic(torch.cat([states,pred],dim=1)).mean()self.actor_optimizer.zero_grad();actor_loss.backward();self.actor_optimizer.step()returncritic_loss.item(),actor_loss.item()

## Augmentation & evaluationâ

importtorchvision.transformsasTclassRobotDataAugmentation:def__init__(self):self.image_aug=T.Compose([T.ColorJitter(0.2,0.2,0.2,0.1),T.RandomRotation(5),T.RandomResizedCrop(224,scale=(0.9,1.0)),T.RandomHorizontalFlip(0.1),])defaugment_episode(self,ep):out=ep.copy()forcam,imgsinep['images'].items():out['images'][cam]=[self.image_aug(img)iftorch.rand(1)<0.5elseimgforimginimgs]iftorch.rand(1)<0.3:out['actions']=ep['actions']+torch.randn_like(ep['actions'])*0.01returnout

importtorchvision.transformsasTclassRobotDataAugmentation:def__init__(self):self.image_aug=T.Compose([T.ColorJitter(0.2,0.2,0.2,0.1),T.RandomRotation(5),T.RandomResizedCrop(224,scale=(0.9,1.0)),T.RandomHorizontalFlip(0.1),])defaugment_episode(self,ep):out=ep.copy()forcam,imgsinep['images'].items():out['images'][cam]=[self.image_aug(img)iftorch.rand(1)<0.5elseimgforimginimgs]iftorch.rand(1)<0.3:out['actions']=ep['actions']+torch.randn_like(ep['actions'])*0.01returnout

### Metricsâ

defevaluate_model(model,test_dataloader,device):model.eval();total,n=0,0withtorch.no_grad():forbatchintest_dataloader:images=list(batch['images'].values())[0][:,0].permute(0,3,1,2).to(device)states=batch['states'][:,0].to(device)actions=batch['actions'][:,0].to(device)mse=F.mse_loss(model(images,states),actions)total+=mse.item()*len(actions);n+=len(actions)returntotal/n

defevaluate_model(model,test_dataloader,device):model.eval();total,n=0,0withtorch.no_grad():forbatchintest_dataloader:images=list(batch['images'].values())[0][:,0].permute(0,3,1,2).to(device)states=batch['states'][:,0].to(device)actions=batch['actions'][:,0].to(device)mse=F.mse_loss(model(images,states),actions)total+=mse.item()*len(actions);n+=len(actions)returntotal/n

## Summaryâ

- Imitation: learn from expert demos

- Sequence: model temporal dependencies (Transformer)

- Multimodal: combine vision, language, actions (VLA)

- RL: offline pretraining or replay initialization