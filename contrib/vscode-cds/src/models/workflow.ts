// Workflow represents a pipeline based workflow
export interface Workflow {
    id: number;
    name: string;
    description: string;
    icon: string;
    project_id: number;
    project_key: string;
    last_modified: string;
    // groups: Array<GroupPermission>;
    // permission: number;
    // metadata: Map<string, string>;
    // usage: Usage;
    // history_length: number;
    // purge_tags: Array<string>;
    // notifications: Array<WorkflowNotification>;
    // from_repository: string;
    // from_template: string;
    // template_up_to_date: boolean;
    // favorite: boolean;
    // pipelines: { [key: number]: Pipeline; };
    // applications: { [key: number]: Application; };
    // environments: { [key: number]: Environment; };
    // project_integrations: { [key: number]: ProjectIntegration; };
    // hook_models: { [key: number]: WorkflowHookModel; };
    // outgoing_hook_models: { [key: number]: WorkflowHookModel; };
    // labels: Label[];
    workflow_data: WorkflowData;
    // as_code_events: Array<AsCodeEvents>;

    // preview: Workflow;
    // asCode: string;
    // audits: AuditWorkflow[];

    // // UI params
    // externalChange: boolean;
    // forceRefresh: boolean;
    // previewMode: boolean;
}

export interface WorkflowData {
    node: WNode;
    joins: Array<WNode>;
}

export interface WNode {
    id: number;
    workflow_id: number;
    name: string;
    ref: string;
    type: string;
    triggers: Array<WNodeTrigger>;
    context: WNodeContext;
    outgoing_hook: WNodeOutgoingHook;
    parents: Array<WNodeJoin>;
    hooks: Array<WNodeHook>;
 //   groups: Array<GroupPermission>;
}

export interface WNodeTrigger {
    id: number;
    parent_node_id: number;
    child_node_id: number;
    parent_node_name: string;
    child_node: WNode;
}

export interface WNodeContext {
    id: number;
    node_id: number;
    pipeline_id: number;
    application_id: number;
    disable_vcs_status: boolean;
    environment_id: number;
    project_integration_id: number;
    default_payload: {};
    //default_pipeline_parameters: Array<Parameter>;
    //conditions: WorkflowNodeConditions;
    mutex: boolean;
}

export interface WNodeOutgoingHook {
    id: number;
    node_id: number;
    hook_model_id: number;
    uuid: string;
    //config: Map<string, WorkflowNodeHookConfigValue>;
}

export interface WNodeJoin {
    id: number;
    node_id: number;
    parent_name: string;
    parent_id: number;
}

export interface WNodeHook {
    id: number;
    uuid: string;
    ref: string;
    node_id: number;
    hook_model_id: number;
    //config: Map<string, WorkflowNodeHookConfigValue>;

    // UI only
    //model: WorkflowHookModel;
}