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
  ReactFlowProvider,
  useReactFlow,
  type NodeProps,
  Handle,
  Position,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import './App.css';
import type { Pipeline, Argument, PortDef } from './types/pipeline';
import * as api from './api';

// ── Types ────────────────────────────────────────────────────

interface RegisteredComponent {
  id: string;
  name: string;
  image: string;
  command: string[];
  args: Argument[];
  cpu: string;
  memory: string;
  disk: string;
  inputPorts?: PortDef[];
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

// ── Custom Pipeline Node ────────────────────────────────────

function PipelineStepNode({ data, selected }: NodeProps) {
  return (
    <div className={`pipeline-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Left} className="node-handle" />
      <div className="node-header">
        <span className="node-status-dot" />
        <span>{data.label as string}</span>
      </div>
      <div className="node-body">
        <div className="node-info">{data.image as string}</div>
      </div>
      <Handle type="source" position={Position.Right} className="node-handle" />
    </div>
  );
}

const nodeTypes: NodeTypes = { pipelineStep: PipelineStepNode };

let nodeCounter = 0;

function createPipelineNode(comp: RegisteredComponent, x: number, y: number): Node {
  nodeCounter++;
  const id = `step-${nodeCounter}`;
  // Pre-fill args from input port default values (F2.10).
  const args: Argument[] = [...(comp.args || [])];
  if (comp.inputPorts) {
    for (const port of comp.inputPorts) {
      if (port.default_value && !args.some((a) => a.name === port.name)) {
        args.push({ name: port.name, value: port.default_value });
      }
    }
  }
  return {
    id,
    type: 'pipelineStep',
    position: { x, y },
    data: {
      label: comp.name,
      image: comp.image,
      command: comp.command,
      args,
      cpu: comp.cpu,
      memory: comp.memory,
      disk: comp.disk,
    },
  };
}

// ── Component Manager Panel ────────────────────────────────

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

// ── Palette ────────────────────────────────────────────────

function Palette({ components, onDragStart }: { components: RegisteredComponent[]; onDragStart: (e: DragEvent, c: RegisteredComponent) => void }) {
  return (
    <aside className="palette">
      <div className="palette-header">
        <h3>Components</h3>
        <span className="palette-count">{components.length}</span>
      </div>
      {components.map((c) => (
        <div key={c.id} className="palette-item" draggable onDragStart={(e) => onDragStart(e, c)}>
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

// ── Config Panel ───────────────────────────────────────────

function NodeConfig({ node, onUpdate }: { node: Node; onUpdate: (id: string, data: Record<string, unknown>) => void }) {
  return (
    <>
      <div className="config-panel-header">
        <h3>Node Config</h3>
      </div>
      <div className="config-content">
        <div className="config-field">
          <label>Name</label>
          <input value={(node.data.label as string) || ''} onChange={(e) => onUpdate(node.id, { label: e.target.value })} />
        </div>
        <div className="config-field">
          <label>Image</label>
          <input value={(node.data.image as string) || ''} onChange={(e) => onUpdate(node.id, { image: e.target.value })} />
        </div>
        <div className="config-field">
          <label>Command (JSON)</label>
          <input value={JSON.stringify((node.data.command as string[]) || [])} onChange={(e) => { try { onUpdate(node.id, { command: JSON.parse(e.target.value) }); } catch {} }} />
        </div>
        <div className="config-section-title">Resources</div>
        <div className="config-field">
          <label>CPU</label>
          <input value={(node.data.cpu as string) || ''} onChange={(e) => onUpdate(node.id, { cpu: e.target.value })} placeholder="500m" />
        </div>
        <div className="config-field">
          <label>Memory</label>
          <input value={(node.data.memory as string) || ''} onChange={(e) => onUpdate(node.id, { memory: e.target.value })} placeholder="256Mi" />
        </div>
        <div className="config-field">
          <label>Disk</label>
          <input value={(node.data.disk as string) || ''} onChange={(e) => onUpdate(node.id, { disk: e.target.value })} placeholder="1Gi" />
        </div>
      </div>
    </>
  );
}

// ── Pipeline Canvas ────────────────────────────────────────

function PipelineCanvas() {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const reactFlow = useReactFlow();
  const [pipelineName, setPipelineName] = useState('my-pipeline');
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [jsonOutput, setJsonOutput] = useState<string | null>(null);
  const [registeredComponents, setRegisteredComponents] = useState<RegisteredComponent[]>(loadComponents);
  const [view, setView] = useState<'pipeline' | 'components' | 'deployments'>('pipeline');
  const [deployDialog, setDeployDialog] = useState<{ open: boolean; deploying: boolean; done: boolean; name: string; result?: api.Deployment; error?: string }>({ open: false, deploying: false, done: false, name: '' });
  const [deployments, setDeployments] = useState<api.Deployment[]>([]);
  const [templates, setTemplates] = useState<api.PipelineTemplate[]>([]);

  useEffect(() => { saveComponents(registeredComponents); }, [registeredComponents]);

  const onConnect = useCallback(
    (connection: Connection) => setEdges((eds) => addEdge(connection, eds)),
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
        const newNode = createPipelineNode(comp, position.x, position.y);
        setNodes((nds) => nds.concat(newNode));
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
    const pipeline: Pipeline = {
      name: pipelineName,
      version: '1',
      nodes: nodes.map((n) => {
        const d = n.data as Record<string, unknown>;
        return {
          id: n.id,
          component: {
            name: (d.label as string) || '',
            image: (d.image as string) || '',
            command: (d.command as string[]) || [],
            args: (d.args as Argument[]) || [],
            resources: d.cpu || d.memory || d.disk ? { cpu: d.cpu as string, memory: d.memory as string, disk: d.disk as string } : undefined,
          },
          outputs: [],
        };
      }),
      edges: edges.map((e) => ({ source: e.source, target: e.target })),
    };
    setJsonOutput(JSON.stringify(pipeline, null, 2));
  }, [nodes, edges, pipelineName]);

  const importPipeline = useCallback(() => {
    const text = prompt('Paste pipeline JSON:');
    if (!text) return;
    try {
      const pipeline: Pipeline = JSON.parse(text);
      setNodes(pipeline.nodes.map((pn, i) => ({
        id: pn.id,
        type: 'pipelineStep' as const,
        position: { x: 100 + i * 50, y: 100 + i * 80 },
        data: { label: pn.component.name, image: pn.component.image, command: pn.component.command || [], args: pn.component.args || [], cpu: pn.component.resources?.cpu || '', memory: pn.component.resources?.memory || '', disk: pn.component.resources?.disk || '' },
      })));
      setEdges(pipeline.edges.map((pe, i) => ({ id: `e-${i}`, source: pe.source, target: pe.target })));
      setJsonOutput(null);
    } catch { alert('Invalid JSON'); }
  }, [setNodes, setEdges]);

  const clearCanvas = useCallback(() => {
    setNodes([]);
    setEdges([]);
    setSelectedNode(null);
    setJsonOutput(null);
  }, [setNodes, setEdges]);

  // ── Deploy ──────────────────────────────────────

  const openDeployDialog = useCallback(() => {
    setDeployDialog({ open: true, deploying: false, done: false, name: pipelineName });
  }, [pipelineName]);

  const closeDeployDialog = useCallback(() => {
    setDeployDialog({ open: false, deploying: false, done: false, name: '', result: undefined, error: undefined });
  }, []);

  const handleDeploy = useCallback(async () => {
    setDeployDialog(prev => ({ ...prev, deploying: true, done: false, error: undefined }));
    try {
      const pipeline: Pipeline = {
        name: pipelineName,
        version: '1',
        nodes: nodes.map((n) => {
          const d = n.data as Record<string, unknown>;
          return {
            id: n.id,
            component: {
              name: (d.label as string) || '',
              image: (d.image as string) || '',
              command: (d.command as string[]) || [],
              args: (d.args as Argument[]) || [],
              resources: d.cpu || d.memory || d.disk ? { cpu: d.cpu as string, memory: d.memory as string, disk: d.disk as string } : undefined,
            },
            outputs: [],
          };
        }),
        edges: edges.map((e) => ({ source: e.source, target: e.target })),
      };
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
    if (view === 'deployments') refreshDeployments();
  }, [view, refreshDeployments]);

  // Auto-refresh deployments every 10s when on deployments tab
  useEffect(() => {
    if (view !== 'deployments') return;
    const id = setInterval(refreshDeployments, 10000);
    return () => clearInterval(id);
  }, [view, refreshDeployments]);

  const handleSaveTemplate = useCallback(async () => {
    const name = prompt('Pipeline template name:', pipelineName);
    if (!name) return;
    const pipeline: Pipeline = {
      name: pipelineName,
      version: '1',
      nodes: nodes.map((n) => {
        const d = n.data as Record<string, unknown>;
        return {
          id: n.id,
          component: {
            name: (d.label as string) || '',
            image: (d.image as string) || '',
            command: (d.command as string[]) || [],
            args: (d.args as Argument[]) || [],
            resources: d.cpu || d.memory || d.disk ? { cpu: d.cpu as string, memory: d.memory as string, disk: d.disk as string } : undefined,
          },
          outputs: [],
        };
      }),
      edges: edges.map((e) => ({ source: e.source, target: e.target })),
    };
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

  const handleDeleteTemplate = useCallback(async (id: string) => {
    try {
      await api.deletePipeline(id);
      refreshDeployments();
    } catch { /* ignore */ }
  }, []);

  return (
    <div className="app">
      <header className="app-header">
        <h1>DataBrew Pipeline</h1>
        <nav className="tab-nav">
          <button className={`tab-btn ${view === 'pipeline' ? 'active' : ''}`} onClick={() => setView('pipeline')}>Pipeline</button>
          <button className={`tab-btn ${view === 'components' ? 'active' : ''}`} onClick={() => setView('components')}>Registry</button>
          <button className={`tab-btn ${view === 'deployments' ? 'active' : ''}`} onClick={() => setView('deployments')}>Deploy</button>
        </nav>
        {view === 'pipeline' && (
          <>
            <input className="pipeline-name-input" value={pipelineName} onChange={(e) => setPipelineName(e.target.value)} placeholder="pipeline-name" />
            <div className="toolbar">
              <button className="terminal-btn primary" onClick={openDeployDialog}>Deploy</button>
              <button className="terminal-btn" onClick={handleSaveTemplate}>Save</button>
              <button className="terminal-btn" onClick={exportPipeline}>Export</button>
              <button className="terminal-btn" onClick={importPipeline}>Import</button>
              <button className="terminal-btn danger" onClick={clearCanvas}>Clear</button>
            </div>
          </>
        )}
      </header>

      <div className="app-body">
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

      {jsonOutput && <pre className="json-output">{jsonOutput}</pre>}

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
                      <button className="terminal-btn" onClick={() => { setView('deployments'); closeDeployDialog(); }}>View Deployments</button>
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

// ── Root App ────────────────────────────────────────────────

export default function App() {
  return (
    <ReactFlowProvider>
      <PipelineCanvas />
    </ReactFlowProvider>
  );
}
