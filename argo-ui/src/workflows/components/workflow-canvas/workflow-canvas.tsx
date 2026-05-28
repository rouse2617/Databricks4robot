import {Page} from 'argo-ui/src/components/page/page';
import * as React from 'react';
import {useState} from 'react';
import {RouteComponentProps} from 'react-router';

import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {CyberApi} from '../../../shared/adapters/cyber-api';

export function WorkflowCanvas({history, match}: RouteComponentProps<any>) {
    const [namespace, setNamespace] = useState(match.params.namespace || '');
    const [name, setName] = useState('');
    const [error, setError] = useState<Error>();
    const [submitting, setSubmitting] = useState(false);

    async function handleSubmit() {
        setSubmitting(true);
        try {
            await CyberApi.deploy({name}, name || undefined);
            history.push(uiUrl(`workflows/${namespace}`));
        } catch (err) {
            setError(err);
            setSubmitting(false);
        }
    }

    return (
        <Page
            title='Canvas'
            toolbar={{
                breadcrumbs: [
                    {title: 'Workflows', path: uiUrl('workflows')},
                    {title: 'Canvas', path: uiUrl('workflows/new')}
                ]
            }}>
            <div className='row'>
                <div className='columns small-12'>
                    <h4>Pipeline Canvas</h4>
                    <p>Design your pipeline using the drag-and-drop canvas below.</p>
                </div>
            </div>
            <ErrorNotice error={error} />
            <div
                style={{
                    border: '2px dashed #ccc',
                    borderRadius: 8,
                    minHeight: 400,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    marginBottom: 16
                }}>
                <div style={{textAlign: 'center', color: '#999'}}>
                    <i className='fa fa-object-group' style={{fontSize: 48, display: 'block', marginBottom: 8}} />
                    <span>Canvas Area</span>
                </div>
            </div>
            <div className='white-box'>
                <div className='row'>
                    <div className='columns small-4'>
                        <label>Namespace</label>
                        <input
                            className='argo-field'
                            value={namespace}
                            onChange={e => setNamespace(e.target.value)}
                            placeholder='default'
                        />
                    </div>
                    <div className='columns small-4'>
                        <label>Workflow Name</label>
                        <input
                            className='argo-field'
                            value={name}
                            onChange={e => setName(e.target.value)}
                            placeholder='my-workflow'
                        />
                    </div>
                </div>
                <div style={{marginTop: 16}}>
                    <button onClick={handleSubmit} className='argo-button argo-button--base' disabled={submitting}>
                        <i className='fa fa-plus' /> {submitting ? 'Submitting...' : 'Submit'}
                    </button>
                </div>
            </div>
        </Page>
    );
}
