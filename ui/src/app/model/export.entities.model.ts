import { WorkflowNodeConditions } from 'app/model/workflow.model';
import * as jsyaml from 'js-yaml';

export class WorkflowEntry {
    name: string;
    description: string;
    version: string;
    template: string;
    workflow: {[key: string]: NodeEntry};
    hooks: {[key: string]: Array<HookEntry>};

    constructor() {
        this.name = 'workflow-name';
        this.description = 'my description';
        this.version = 'v1.0';
        this.template = '';
        this.workflow = {
            'fake-name': new NodeEntry(),
        };
        this.hooks = null;
    }

    toSnippet(): string {
        return jsyaml.dump(this);
    }
}

export class HookEntry {
    model: string;
    ref: string;
    config: {[key: string]: string};
    conditions: string;

    constructor() {
        this.model = '';
        this.ref = '';
        this.config = null;
        this.conditions = null;
    }

    toSnippet() {
        let snippet = jsyaml.dump(this, {noRefs: true});
        let nodeSnippetLines = snippet.split('\n').map(line => {
            return '    ' + line;
        });
        nodeSnippetLines.unshift('  fake-node:');
        return nodeSnippetLines.join('\n');
    }
}

export class NodeEntry {
    pipeline: string;
    application: string;
    environment: string;
    integration: string;
    depends_on: Array<string>;
    conditions: WorkflowNodeConditions;
    when: Array<string>;
    one_at_a_time: boolean;
    payload: {[key: number]: string};
    parameters: {[key: number]: string};
    trigger: string;
    config: {[key: number]: string};

    constructor() {
        this.pipeline = '';
        this.application = '';
        this.environment = '';
        this.integration = '';
        this.depends_on = null;
        this.conditions = null;
        this.when = null;
        this.one_at_a_time = false;
        this.payload = null;
        this.parameters = null;
        this.trigger = '';
        this.config = null;
    }

    toSnippet() {
        let nodeSnippet =  jsyaml.dump(this, {noRefs: true});
        let nodeSnippetLines = nodeSnippet.split('\n').map(line => {
            return '    ' + line;
        });
        nodeSnippetLines.unshift('  fake-node:');
        return nodeSnippetLines.join('\n');
    }
}
