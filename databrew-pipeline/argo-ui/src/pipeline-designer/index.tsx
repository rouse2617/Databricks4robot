import {ReactFlowProvider} from '@xyflow/react';
import {PipelineCanvas} from './PipelineCanvas';
import './pipeline-designer.scss';

const PipelineDesigner = () => (
  <ReactFlowProvider>
    <PipelineCanvas />
  </ReactFlowProvider>
);

export default { component: PipelineDesigner };
