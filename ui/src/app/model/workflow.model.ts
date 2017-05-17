import {Pipeline} from './pipeline.model';
import {Application} from './application.model';
import {Environment} from './environment.model';

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

    // UI params
    externalChange: boolean;

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
}

export class WorkflowNodeJoinTrigger {
    id: number;
    join_id: number;
    workflow_dest_node_id: number;
    workflow_dest_node: WorkflowNode;
    conditions: Array<WorkflowTriggerCondition>;
}

// WorkflowNode represents a node in w workflow tree
export class WorkflowNode {
    id: number;
    ref: string;
    workflow_id: number;
    pipeline_id: number;
    pipeline: Pipeline;
    context: WorkflowNodeContext;
    hooks: Array<WorkflowNodeHook>;
    triggers: Array<WorkflowNodeTrigger>;
}

// WorkflowNodeContext represents a context attached on a node
export class WorkflowNodeContext {
    id: number;
    workflow_node_id: number;
    application_id: number;
    application: Application;
    environment: Environment;
    environment_id: number;
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
