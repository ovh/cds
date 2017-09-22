import {Pipeline} from './pipeline.model';
import {Application} from './application.model';
import {Environment} from './environment.model';
import {intersection} from 'lodash';
import {Parameter} from './parameter.model';

// Workflow represents a pipeline based workflow
export class Workflow {
    id: number;
    name: string;
    description: string;
    project_id: number;
    project_key: string;
    root: WorkflowNode;
    root_id: number;
    joins: Array<WorkflowNodeJoin>;
    last_modified: Date;

    // UI params
    externalChange: boolean;

    static getNodeByID(id: number, w: Workflow): WorkflowNode {
        let node = WorkflowNode.getNodeByID(w.root, id);
        if (!node && w.joins) {
            quit: for (let i = 0; i < w.joins.length; i++) {
                if (w.joins[i].triggers) {
                    for (let j = 0; j < w.joins[i].triggers.length; j++) {
                        node = WorkflowNode.getNodeByID(w.joins[i].triggers[j].workflow_dest_node, id);
                        if (node) {
                            break quit;
                        }
                    }
                }
            }
        }
        return node;
    }

    static removeOldRef(w: Workflow) {
        if (!w.joins) {
            return;
        }
        let refs = new Array<string>();
        WorkflowNode.addRef(refs, w.root);

        w.joins.forEach(j => {
            if (j.triggers) {
                j.triggers.forEach(t => {
                    WorkflowNode.addRef(refs, t.workflow_dest_node);
                });
            }
        });

        w.joins.forEach(j => {
            j.source_node_ref = intersection(j.source_node_ref, refs);
        });
    }

    constructor() {
        this.root = new WorkflowNode();
    }
}

export class WorkflowNodeJoin {
    id: number;
    workflow_id: number;
    source_node_id: Array<number>;
    source_node_ref: Array<string>;
    triggers: Array<WorkflowNodeJoinTrigger>;

    constructor() {
        this.source_node_ref = new Array<string>();
    }
}

export class WorkflowNodeJoinTrigger {
    id: number;
    join_id: number;
    workflow_dest_node_id: number;
    workflow_dest_node: WorkflowNode;
    conditions: Array<WorkflowTriggerCondition>;
    manual: boolean;

    constructor() {
        this.workflow_dest_node = new WorkflowNode();
    }
}

// WorkflowNode represents a node in w workflow tree
export class WorkflowNode {
    id: number;
    name: string;
    ref: string;
    workflow_id: number;
    pipeline_id: number;
    pipeline: Pipeline;
    context: WorkflowNodeContext;
    hooks: Array<WorkflowNodeHook>;
    triggers: Array<WorkflowNodeTrigger>;

    static getNodeByID(node: WorkflowNode, id: number) {
        if (node.id === id) {
            return node;
        }
        let nodeToFind: WorkflowNode;
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let n = WorkflowNode.getNodeByID(node.triggers[i].workflow_dest_node, id);
                if (n) {
                    nodeToFind = n;
                    break;
                }
            }
        }
        return nodeToFind;
    }

    static addRef(refs: string[], root: WorkflowNode) {
        refs.push(root.ref);
        if (root.triggers) {
            root.triggers.forEach(t => {
                WorkflowNode.addRef(refs, t.workflow_dest_node);
            });
        }
    }

    constructor() {
        this.context = new WorkflowNodeContext();
    }
}

// WorkflowNodeContext represents a context attached on a node
export class WorkflowNodeContext {
    id: number;
    workflow_node_id: number;
    application_id: number;
    application: Application;
    environment: Environment;
    environment_id: number;
    default_payload: {};
    default_pipeline_parameters: Array<Parameter>;
}

// WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
export class WorkflowNodeHook {
    id: number;
    uuid: string;
    model: WorkflowHookModel;
    conditions: Array<WorkflowTriggerCondition>;
    config: {};
}

// WorkflowNodeTrigger is a ling betweeb two pipelines in a workflow
export class WorkflowNodeTrigger {
    id: number;
    workflow_node_id: number;
    workflow_dest_node_id: number;
    workflow_dest_node: WorkflowNode;
    conditions: Array<WorkflowTriggerCondition>;
    manual: boolean;

    constructor() {
        this.workflow_dest_node = new WorkflowNode();
    }
}

// WorkflowTriggerCondition represents a condition to trigger ot not a pipeline in a workflow. Operator can be =, !=, regex
export class WorkflowTriggerCondition {
    variable: string;
    operator: string;
    value: string;
}

export class WorkflowHookModel {
    id: number;
    name: string;
    type: string;
    images: string;
    command: string;
    default_config: {};
}

export class WorkflowTriggerConditionCache {
    operators: Array<string>;
    names: Array<string>;
}
