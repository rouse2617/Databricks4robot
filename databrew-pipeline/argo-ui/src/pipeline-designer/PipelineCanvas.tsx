import { useState, useCallback, useRef, useEffect, type DragEvent } from 'react';
import {
  ReactFlow,
  addEdge,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type Connection,
  type NodeTypes,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  useReactFlow,
  type NodeProps,
  Handle,
  Position,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from 'dagre';
import type { Pipeline, PipelineNode, Argument } from './types';
import * as api from './api';
import { NodeConfig } from './NodeConfigPanel';
import {
  DEFAULT_INPUT_PORT,
  DEFAULT_OUTPUT_PORT,
  buildPipeline,
  defaultInputs,
  defaultOutputs,
  dropOffset,
  pipelineEdgeToFlowEdge,
  pipelineNodeToFlowNode,
} from './pipelineModel';

// ── Registered Component Types ─────────────────────────────

interface RegisteredComponent {
  id: string;
  name: string;
  image: string;
  command: string[];
  args: Argument[];
  cpu: string;
  memory: string;
  disk: string;
}

const STORAGE_KEY = 'databrew-components';

function loadComponents(): RegisteredComponent[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : getDefaultComponents();
  } catch {
    return getDefaultComponents();
  }
}

function saveComponents(comps: RegisteredComponent[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(comps));
}

function getDefaultComponents(): RegisteredComponent[] {
  return [
    { id: 'c1', name: 'BusyBox', image: 'busybox:latest', command: ['sh', '-c'], args: [{ name: 'script', value: 'echo hello' }], cpu: '', memory: '', disk: '' },
    { id: 'c2', name: 'Python', image: 'python:3.12-slim', command: ['python', '-c'], args: [{ name: 'script', value: 'print("hello")' }], cpu: '', memory: '', disk: '' },
    { id: 'c3', name: 'Alpine', image: 'alpine:latest', command: ['sh', '-c'], args: [{ name: 'script', value: 'echo hello' }], cpu: '', memory: '', disk: '' },
  ];
}

// ── Pipeline Step Node (ReactFlow custom node) ─────────────

function PipelineStepNode({ data, selected }: NodeProps) {
  const inputs = (data.inputs as { name: string }[])?.length ? (data.inputs as { name: string }[]) : defaultInputs();
  const outputs = (data.outputs as { name: string }[])?.length ? (data.outputs as { name: string }[]) : defaultOutputs();

  return (
    <div className={`pipeline-node ${selected ? 'selected' : ''}`} style={{ minHeight: Math.max(72, (Math.max(inputs.length, outputs.length) + 1) * 28) }}>
      {inputs.map((p, i) => (
        <Handle
          key={`in-${p.name}`}
          type="target"
          position={Position.Left}
          id={p.name}
          className="node-handle"
          style={{ top: `${((i + 1) / (inputs.length + 1)) * 100}%` }}
        />
      ))}
      <div className="node-header">
        <span className="node-status-dot" />
        <span>{data.label as string}</span>
      </div>
      <div className="node-body">
        <div className="node-info">{data.image as string}</div>
      </div>
      {outputs.map((p, i) => (
        <Handle
          key={`out-${p.name}`}
          type="source"
          position={Position.Right}
          id={p.name}
          className="node-handle"
          style={{ top: `${((i + 1) / (outputs.length + 1)) * 100}%` }}
        />
      ))}
    </div>
  );
}

const nodeTypes: NodeTypes = { pipelineStep: PipelineStepNode };

let nodeCounter = 0;

function createPipelineNode(comp: RegisteredComponent, x: number, y: number, existingCount: number): Node {
  nodeCounter++;
  const id = `step-${nodeCounter}`;
  const { dx, dy } = dropOffset(existingCount);
  return {
    id,
    type: 'pipelineStep',
    position: { x: x + dx, y: y + dy },
    data: {
      label: comp.name,
      image: comp.image,
      command: comp.command,
      args: comp.args || [],
      cpu: comp.cpu,
      memory: comp.memory,
      disk: comp.disk,
      inputs: defaultInputs(),
      outputs: defaultOutputs(),
    },
  };
}

// ── Component Manager Panel ───────────────────────────────

function ComponentManager({ components, onChange }: { components: RegisteredComponent[]; onChange: (c: RegisteredComponent[]) => void }) {
  const [editing, setEditing] = useState<RegisteredComponent | null>(null);
  const [isNew, setIsNew] = useState(false);

  const blank = (): RegisteredComponent => ({
    id: 'c' + Date.now(),
    name: '',
    image: '',
    command: ['sh', '-c'],
    args: [],
    cpu: '',
    memory: '',
    disk: '',
  });

  const save = (c: RegisteredComponent) => {
    if (isNew) {
      onChange([...components, c]);
    } else {
      onChange(components.map((x) => (x.id === c.id ? c : x)));
    }
    setEditing(null);
  };

  const remove = (id: string) => {
    onChange(components.filter((x) => x.id !== id));
    if (editing?.id === id) setEditing(null);
  };

  return (
    <div className="component-manager">
      <div className="cm-header">
        <h3>Component Registry</h3>
        <button className="terminal-btn primary" onClick={() => { setEditing(blank()); setIsNew(true); }}>+ New</button>
      </div>

      <div className="cm-list">
        {components.length === 0 && (
          <div className="config-empty">No components registered</div>
        )}
        {components.map((c) => (
          <div
            key={c.id}
            className={`cm-item ${editing?.id === c.id ? 'active' : ''}`}
            onClick={() => { setEditing(c); setIsNew(false); }}
          >
            <div className="cm-item-name">{c.name}</div>
            <div className="cm-item-image">{c.image}</div>
          </div>
        ))}
      </div>

      {editing && (
        <div className="cm-form">
          <h4>{isNew ? 'New Component' : 'Edit Component'}</h4>
          <div className="cm-form-fields">
            <label>
              Name
              <input value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} placeholder="component-name" />
            </label>
            <label>
              Image
              <input value={editing.image} onChange={(e) => setEditing({ ...editing, image: e.target.value })} placeholder="repo/image:tag" />
            </label>
            <label>
              Command (JSON)
              <input value={JSON.stringify(editing.command)} onChange={(e) => { try { setEditing({ ...editing, command: JSON.parse(e.target.value) }); } catch {} }} placeholder='["sh", "-c"]' />
            </label>
            <label>
              CPU
              <input value={editing.cpu} onChange={(e) => setEditing({ ...editing, cpu: e.target.value })} placeholder="500m, 2" />
            </label>
            <label>
              Memory
              <input value={editing.memory} onChange={(e) => setEditing({ ...editing, memory: e.target.value })} placeholder="256Mi, 1Gi" />
            </label>
            <label>
              Disk
              <input value={editing.disk} onChange={(e) => setEditing({ ...editing, disk: e.target.value })} placeholder="1Gi" />
            </label>
          </div>
          <div className="cm-form-actions">
            <button className="terminal-btn primary" onClick={() => save(editing)}>{isNew ? 'Create' : 'Save'}</button>
            {!isNew && <button className="terminal-btn danger" onClick={() => remove(editing.id)}>Delete</button>}
            <button className="terminal-btn" onClick={() => setEditing(null)}>Cancel</button>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Palette ───────────────────────────────────────────────

function Palette({ components, onDragStart }: { components: RegisteredComponent[]; onDragStart: (e: DragEvent, c: RegisteredComponent) => void }) {
  return (
    <aside className="palette">
      <div className="palette-header">
        <h3>Components</h3>
        <span className="palette-count">{components.length}</span>
      </div>
      {components.map((c) => (
        <div key={c.id} className="palette-item" draggable onDragStart={(e) => onDragStart(e, c)}>
          <span className="pi-drag-indicator">&#x2BFF;</span>
          <div className="pi-content">
            <div className="pi-label">{c.name}</div>
            <div className="pi-image">{c.image}</div>
          </div>
        </div>
      ))}
      {components.length === 0 && (
        <div className="config-empty" style={{ marginTop: 20 }}>No components</div>
      )}
    </aside>
  );
}

// ── Pipeline Canvas ────────────────────────────────────────

export function PipelineCanvas() {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const reactFlow = useReactFlow();
  const [pipelineName, setPipelineName] = useState('my-pipeline');
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [jsonOutput, setJsonOutput] = useState<string | null>(null);
  const [registeredComponents, setRegisteredComponents] = useState<RegisteredComponent[]>(loadComponents);
  const [view, setView] = useState<'pipeline' | 'components' | 'deploy'>('pipeline');
  const [deployDialog, setDeployDialog] = useState<{ open: boolean; deploying: boolean; done: boolean; name: string; result?: api.Deployment; error?: string }>({ open: false, deploying: false, done: false, name: '' });
  const [deployments, setDeployments] = useState<api.Deployment[]>([]);
  const [templates, setTemplates] = useState<api.PipelineTemplate[]>([]);

  useEffect(() => { saveComponents(registeredComponents); }, [registeredComponents]);

  const switchView = useCallback((next: 'pipeline' | 'components' | 'deploy') => {
    if (next !== 'pipeline') setJsonOutput(null);
    setView(next);
  }, []);

  const onConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return;
      setEdges((eds) =>
        addEdge(
          {
            ...connection,
            sourceHandle: connection.sourceHandle ?? DEFAULT_OUTPUT_PORT,
            targetHandle: connection.targetHandle ?? DEFAULT_INPUT_PORT,
          },
          eds,
        ),
      );
    },
    [setEdges],
  );

  const onDragStart = useCallback((e: DragEvent, comp: RegisteredComponent) => {
    e.dataTransfer.setData('application/reactflow', JSON.stringify(comp));
    e.dataTransfer.effectAllowed = 'move';
  }, []);

  const onDragOver = useCallback((event: DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: DragEvent) => {
      event.preventDefault();
      const raw = event.dataTransfer.getData('application/reactflow');
      if (!raw) return;
      try {
        const comp: RegisteredComponent = JSON.parse(raw);
        const bounds = wrapperRef.current?.getBoundingClientRect();
        if (!bounds) return;
        const position = reactFlow.screenToFlowPosition({ x: event.clientX, y: event.clientY });
        setNodes((nds) => {
          const newNode = createPipelineNode(comp, position.x, position.y, nds.length);
          return nds.concat(newNode);
        });
      } catch {
        /* ignore */
      }
    },
    [reactFlow, setNodes],
  );

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => setSelectedNode(node), []);
  const onPaneClick = useCallback(() => setSelectedNode(null), []);

  const updateNodeData = useCallback(
    (id: string, data: Record<string, unknown>) => {
      setNodes((nds) => nds.map((n) => (n.id === id ? { ...n, data: { ...n.data, ...data } } : n)));
      setSelectedNode((prev) => prev?.id === id ? { ...prev, data: { ...prev.data, ...data } } : prev);
    },
    [setNodes],
  );

  const exportPipeline = useCallback(() => {
    const pipeline = buildPipeline(pipelineName, nodes, edges);
    setJsonOutput(JSON.stringify(pipeline, null, 2));
  }, [nodes, edges, pipelineName]);

  const importPipeline = useCallback(() => {
    const text = prompt('Paste pipeline JSON:');
    if (!text) return;
    try {
      const pipeline: Pipeline = JSON.parse(text);
      if (pipeline.name) setPipelineName(pipeline.name);
      setNodes(pipeline.nodes.map((pn, i) => pipelineNodeToFlowNode(pn, i)));
      setEdges(pipeline.edges.map((pe, i) => pipelineEdgeToFlowEdge(pe, i)));
      setJsonOutput(null);
    } catch { alert('Invalid JSON'); }
  }, [setNodes, setEdges]);

  const clearCanvas = useCallback(() => {
    setNodes([]);
    setEdges([]);
    setSelectedNode(null);
    setJsonOutput(null);
  }, [setNodes, setEdges]);

  const autoLayout = useCallback(() => {
    setNodes((nds) => {
      if (nds.length === 0) return nds;
      const g = new dagre.graphlib.Graph();
      g.setDefaultEdgeLabel(() => ({}));
      g.setGraph({ rankdir: 'LR', nodesep: 50, ranksep: 80, marginx: 40, marginy: 40 });

      const nodeWidth = 200;
      const nodeHeight = 80;

      nds.forEach((n) => g.setNode(n.id, { width: nodeWidth, height: nodeHeight }));
      setEdges((eds) => {
        eds.forEach((e) => g.setEdge(e.source, e.target));
        return eds;
      });
      dagre.layout(g);

      return nds.map((n) => {
        const dagreNode = g.node(n.id);
        if (!dagreNode) return n;
        return {
          ...n,
          position: {
            x: dagreNode.x - nodeWidth / 2,
            y: dagreNode.y - nodeHeight / 2,
          },
        };
      });
    });
  }, [setNodes, setEdges]);

  // ── Deploy ─────────────────────────────────────────────

  const openDeployDialog = useCallback(() => {
    setDeployDialog({ open: true, deploying: false, done: false, name: pipelineName });
  }, [pipelineName]);

  const closeDeployDialog = useCallback(() => {
    setDeployDialog({ open: false, deploying: false, done: false, name: '', result: undefined, error: undefined });
  }, []);

  const handleDeploy = useCallback(async () => {
    setDeployDialog(prev => ({ ...prev, deploying: true, done: false, error: undefined }));
    try {
      const pipeline = buildPipeline(pipelineName, nodes, edges);
      const result = await api.deploy(pipeline, deployDialog.name || pipelineName);
      setDeployDialog(prev => ({ ...prev, deploying: false, done: true, result }));
      refreshDeployments();
    } catch (err) {
      setDeployDialog(prev => ({ ...prev, deploying: false, done: true, error: String(err) }));
    }
  }, [nodes, edges, pipelineName, deployDialog.name]);

  const refreshDeployments = useCallback(async () => {
    try {
      const [d, t] = await Promise.all([api.listDeployments(), api.listPipelines()]);
      setDeployments(d);
      setTemplates(t);
    } catch { /* server not available */ }
  }, []);

  useEffect(() => {
    if (view === 'deploy') refreshDeployments();
  }, [view, refreshDeployments]);

  useEffect(() => {
    if (view !== 'deploy') return;
    const id = setInterval(refreshDeployments, 10000);
    return () => clearInterval(id);
  }, [view, refreshDeployments]);

  const handleSaveTemplate = useCallback(async () => {
    const name = prompt('Pipeline template name:', pipelineName);
    if (!name) return;
    const pipeline = buildPipeline(pipelineName, nodes, edges);
    try {
      await api.savePipeline(name, pipeline);
      refreshDeployments();
    } catch (err) {
      alert('Save failed: ' + String(err));
    }
  }, [nodes, edges, pipelineName]);

  const handleDeployTemplate = useCallback(async (id: string) => {
    try {
      await api.deployTemplate(id);
      refreshDeployments();
    } catch (err) {
      alert('Deploy failed: ' + String(err));
    }
  }, []);

  const handleDeleteDeployment = useCallback(async (id: string) => {
    try {
      await api.deleteDeployment(id);
      refreshDeployments();
    } catch { /* ignore */ }
  }, []);

  const handleLoadTemplate = useCallback(async (id: string) => {
    try {
      const tpl = await api.getPipeline(id);
      const pipeline = tpl.pipeline as Pipeline;
      if (!pipeline?.nodes) { alert('Invalid pipeline data'); return; }
      setPipelineName(pipeline.name || tpl.name);
      setNodes(pipeline.nodes.map((pn: PipelineNode, i: number) => pipelineNodeToFlowNode(pn, i)));
      setEdges(pipeline.edges.map((pe, i) => pipelineEdgeToFlowEdge(pe, i)));
      setView('pipeline');
    } catch (err) {
      alert('Load failed: ' + String(err));
    }
  }, [setNodes, setEdges]);

  const handleDeleteTemplate = useCallback(async (id: string) => {
    try {
      await api.deletePipeline(id);
      refreshDeployments();
    } catch { /* ignore */ }
  }, []);

  return (
    <div className="pipeline-designer">
      {/* ── Header Bar ── */}
      <div className="pd-header">
        <div className="pd-title">Pipeline</div>
        <nav className="pd-tabs">
          <button className={`pd-tab ${view === 'pipeline' ? 'active' : ''}`} onClick={() => switchView('pipeline')}>Pipeline</button>
          <button className={`pd-tab ${view === 'components' ? 'active' : ''}`} onClick={() => switchView('components')}>Registry</button>
          <button className={`pd-tab ${view === 'deploy' ? 'active' : ''}`} onClick={() => switchView('deploy')}>Deploy</button>
        </nav>
        {view === 'pipeline' && (
          <>
            <input
              className="pd-name-input"
              value={pipelineName}
              onChange={(e) => setPipelineName(e.target.value)}
              placeholder="pipeline-name"
            />
            <div className="pd-toolbar">
              <button className="terminal-btn primary" onClick={openDeployDialog}>Deploy</button>
              <button className="terminal-btn" onClick={handleSaveTemplate}>Save</button>
              <button className="terminal-btn" onClick={exportPipeline}>Export</button>
              <button className="terminal-btn" onClick={importPipeline}>Import</button>
              <button className="terminal-btn" onClick={autoLayout}>Layout</button>
              <button className="terminal-btn danger" onClick={clearCanvas}>Clear</button>
            </div>
          </>
        )}
      </div>

      {/* ── Body ── */}
      <div className="pd-body">
        {view === 'pipeline' ? (
          <>
            <Palette components={registeredComponents} onDragStart={onDragStart} />
            <div className="canvas-wrapper" ref={wrapperRef}>
              <ReactFlow
                nodes={nodes} edges={edges}
                onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
                onConnect={onConnect} onDrop={onDrop} onDragOver={onDragOver}
                onNodeClick={onNodeClick} onPaneClick={onPaneClick}
                nodeTypes={nodeTypes} fitView
              >
                <Background variant={BackgroundVariant.Dots} gap={24} color="#d4c9bc" />
                <Controls />
                <MiniMap />
              </ReactFlow>
            </div>
            <aside className="config-panel">
              {selectedNode ? <NodeConfig node={selectedNode} onUpdate={updateNodeData} /> : (
                <div className="config-empty">
                  Select a node to configure
                </div>
              )}
            </aside>
          </>
        ) : view === 'components' ? (
          <div className="registry-view">
            <ComponentManager components={registeredComponents} onChange={setRegisteredComponents} />
          </div>
        ) : (
          <div className="deploy-view">
            <div className="deploy-panel">
              <h3>Deployments</h3>

              <div className="deploy-section">
                <div className="deploy-section-title">
                  Saved Pipelines
                  <span className="count">{templates.length}</span>
                </div>
                {templates.length === 0 ? (
                  <div className="dep-empty">No saved pipelines. Use Save in the Pipeline tab to save one.</div>
                ) : (
                  templates.map((t) => (
                    <div key={t.id} className="dep-card">
                      <div className="dep-card-info">
                        <div className="dep-card-name">{t.name}</div>
                        <div className="dep-card-meta">
                          <span>{t.nodeCount} nodes</span>
                          <span className="dot">•</span>
                          <span>{new Date(t.createdAt).toLocaleString()}</span>
                        </div>
                      </div>
                      <div className="deploy-btn-list">
                        <button className="terminal-btn primary" onClick={() => handleDeployTemplate(t.id)}>Deploy</button>
                        <button className="terminal-btn" onClick={() => handleLoadTemplate(t.id)}>Load</button>
                        <button className="terminal-btn danger" onClick={() => handleDeleteTemplate(t.id)}>Del</button>
                      </div>
                    </div>
                  ))
                )}
              </div>

              <div className="deploy-section-title" style={{ marginTop: 20 }}>
                Deploy History
                <span className="count">{deployments.length}</span>
              </div>
              <div className="deploy-section">
                {deployments.length === 0 ? (
                  <div className="dep-empty">No deployments yet.</div>
                ) : (
                  deployments.map((d) => (
                    <div key={d.id} className="dep-card">
                      <div className="dep-card-info">
                        <div className="dep-card-name">{d.pipelineName}</div>
                        <div className="dep-card-meta">
                          <span className={`status-badge ${(d.status || 'unknown').toLowerCase()}`}>{d.status}</span>
                          <span>{d.nodes} nodes</span>
                          <span className="dot">•</span>
                          <span>{new Date(d.createdAt).toLocaleString()}</span>
                          {d.finishedAt && (
                            <>
                              <span className="dot">•</span>
                              <span>fin: {new Date(d.finishedAt).toLocaleString()}</span>
                            </>
                          )}
                        </div>
                      </div>
                      <div className="deploy-btn-list">
                        {d.status === 'Running' || d.status === 'Pending' ? (
                          <span className="status-badge running">{d.status}</span>
                        ) : (
                          <>
                            {d.status === 'Succeeded' && <span className="status-badge succeeded">Succeeded</span>}
                            {(d.status === 'Failed' || d.status === 'Error') && <span className="status-badge failed">{d.status}</span>}
                          </>
                        )}
                        <button className="terminal-btn danger" onClick={() => handleDeleteDeployment(d.id)}>Del</button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}
      </div>

      {view === 'pipeline' && jsonOutput && <pre className="json-output">{jsonOutput}</pre>}

      {/* ── Deploy Dialog ── */}
      {deployDialog.open && (
        <div className="deploy-overlay" onClick={closeDeployDialog}>
          <div className="deploy-dialog" onClick={(e) => e.stopPropagation()}>
            {!deployDialog.deploying && !deployDialog.done && (
              <>
                <h3>Deploy Pipeline</h3>
                <p style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 14 }}>
                  This will transpile the pipeline and submit it to the Kubernetes cluster.
                </p>
                <div className="deploy-dialog-fields">
                  <label>
                    Workflow Name
                    <input
                      value={deployDialog.name}
                      onChange={(e) => setDeployDialog(prev => ({ ...prev, name: e.target.value }))}
                      placeholder={pipelineName}
                    />
                  </label>
                  <div className="dep-card-meta" style={{ fontSize: 11 }}>
                    <span>{nodes.length} node(s)</span>
                    <span className="dot">•</span>
                    <span>{edges.length} edge(s)</span>
                  </div>
                </div>
                <div className="deploy-dialog-actions">
                  <button className="terminal-btn" onClick={closeDeployDialog}>Cancel</button>
                  <button className="terminal-btn primary" onClick={handleDeploy} disabled={nodes.length === 0}>Deploy</button>
                </div>
              </>
            )}
            {deployDialog.deploying && (
              <div className="deploy-progress">
                <div className="deploy-spinner" />
                <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Deploying pipeline...</p>
              </div>
            )}
            {deployDialog.done && (
              <div className="deploy-result">
                {deployDialog.error ? (
                  <>
                    <div className="status-badge failed">Failed</div>
                    <p style={{ fontSize: 12, color: 'var(--danger)', marginTop: 8, fontFamily: 'var(--font-mono)', whiteSpace: 'pre-wrap' }}>{deployDialog.error}</p>
                  </>
                ) : deployDialog.result ? (
                  <>
                    <div className="status-badge succeeded">Deployed</div>
                    <p className="wf-name">{deployDialog.result.workflowName}</p>
                    <div className="dep-card-meta" style={{ justifyContent: 'center', marginTop: 8 }}>
                      <span>{deployDialog.result.nodes} nodes</span>
                      <span className="dot">•</span>
                      <span>{new Date(deployDialog.result.createdAt).toLocaleString()}</span>
                    </div>
                    <div style={{ marginTop: 16 }}>
                      <button className="terminal-btn" onClick={() => { switchView('deploy'); closeDeployDialog(); }}>View Deployments</button>
                      <button className="terminal-btn" onClick={closeDeployDialog} style={{ marginLeft: 8 }}>Close</button>
                    </div>
                  </>
                ) : null}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
