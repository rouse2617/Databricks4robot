import type { Node } from '@xyflow/react';
import type { Argument, EnvVar, Port } from './types';

function parsePortsJson(text: string): Port[] | null {
  try {
    const parsed = JSON.parse(text) as Port[];
    if (!Array.isArray(parsed)) return null;
    return parsed.filter((p) => p?.name?.trim());
  } catch {
    return null;
  }
}

function parseArgsJson(text: string): Argument[] | null {
  try {
    const parsed = JSON.parse(text) as Argument[];
    if (!Array.isArray(parsed)) return null;
    return parsed;
  } catch {
    return null;
  }
}

function parseEnvJson(text: string): EnvVar[] | null {
  try {
    const parsed = JSON.parse(text) as EnvVar[];
    if (!Array.isArray(parsed)) return null;
    return parsed.filter((e) => e?.name?.trim());
  } catch {
    return null;
  }
}

export function NodeConfig({ node, onUpdate }: { node: Node; onUpdate: (id: string, data: Record<string, unknown>) => void }) {
  const inputs = (node.data.inputs as Port[]) || [];
  const outputs = (node.data.outputs as Port[]) || [];
  const args = (node.data.args as Argument[]) || [];
  const envVars = (node.data.env as EnvVar[]) || [];

  return (
    <>
      <div className="config-panel-header">
        <h3>Node Config</h3>
      </div>
      <div className="config-content">
        <div className="config-field">
          <label>Name</label>
          <input
            value={(node.data.label as string) || ''}
            onChange={(e) => onUpdate(node.id, { label: e.target.value })}
          />
        </div>
        <div className="config-field">
          <label>Image</label>
          <input
            value={(node.data.image as string) || ''}
            onChange={(e) => onUpdate(node.id, { image: e.target.value })}
          />
        </div>
        <div className="config-field">
          <label>Image Pull Policy</label>
          <select
            value={(node.data.imagePullPolicy as string) || ''}
            onChange={(e) => onUpdate(node.id, { imagePullPolicy: e.target.value })}
          >
            <option value="">(default)</option>
            <option value="Always">Always</option>
            <option value="IfNotPresent">IfNotPresent</option>
            <option value="Never">Never</option>
          </select>
        </div>
        <div className="config-field">
          <label>Command (JSON)</label>
          <input
            value={JSON.stringify((node.data.command as string[]) || [])}
            onChange={(e) => {
              try {
                onUpdate(node.id, { command: JSON.parse(e.target.value) });
              } catch {
                /* invalid json */
              }
            }}
          />
        </div>
        <div className="config-field">
          <label>Args (JSON)</label>
          <textarea
            className="config-textarea"
            rows={4}
            value={JSON.stringify(args, null, 2)}
            onChange={(e) => {
              const parsed = parseArgsJson(e.target.value);
              if (parsed) onUpdate(node.id, { args: parsed });
            }}
            placeholder='[{"name":"script","value":"echo hi"}]'
          />
        </div>
        <div className="config-section-title">Environment</div>
        <div className="config-field">
          <label>Env Vars (JSON)</label>
          <textarea
            className="config-textarea"
            rows={4}
            value={JSON.stringify(envVars, null, 2)}
            onChange={(e) => {
              const parsed = parseEnvJson(e.target.value);
              if (parsed) onUpdate(node.id, { env: parsed });
            }}
            placeholder='[{"name":"API_KEY","value":"secret123"}]'
          />
        </div>
        <div className="config-section-title">Ports</div>
        <div className="config-field">
          <label>Inputs (JSON)</label>
          <textarea
            className="config-textarea"
            rows={3}
            value={JSON.stringify(inputs, null, 2)}
            onChange={(e) => {
              const parsed = parsePortsJson(e.target.value);
              if (parsed?.length) onUpdate(node.id, { inputs: parsed });
            }}
            placeholder='[{"name":"in","type":"string"}]'
          />
        </div>
        <div className="config-field">
          <label>Outputs (JSON)</label>
          <textarea
            className="config-textarea"
            rows={3}
            value={JSON.stringify(outputs, null, 2)}
            onChange={(e) => {
              const parsed = parsePortsJson(e.target.value);
              if (parsed?.length) onUpdate(node.id, { outputs: parsed });
            }}
            placeholder='[{"name":"out","type":"string"}]'
          />
        </div>
        <div className="config-section-title">Resources</div>
        <div className="config-field">
          <label>CPU</label>
          <input
            value={(node.data.cpu as string) || ''}
            onChange={(e) => onUpdate(node.id, { cpu: e.target.value })}
            placeholder="500m"
          />
        </div>
        <div className="config-field">
          <label>Memory</label>
          <input
            value={(node.data.memory as string) || ''}
            onChange={(e) => onUpdate(node.id, { memory: e.target.value })}
            placeholder="256Mi"
          />
        </div>
        <div className="config-field">
          <label>Disk</label>
          <input
            value={(node.data.disk as string) || ''}
            onChange={(e) => onUpdate(node.id, { disk: e.target.value })}
            placeholder="1Gi"
          />
        </div>
      </div>
    </>
  );
}
