export class Event {
    healthCheck: number;
    timestamp: Date;
    hostname: string;
    cdsname: string;
    type_event: string;
    payload: {};
    attempt: number;
    username: string;
    user_mail: string;
    project_key: string;
    application_name: string;
    pipeline_name: string;
    environment_name: string;
    workflow_name: string;
    status: string;
    workflow_run_num: number;
    workflow_run_num_sub: number;
}

export class EventType {
    static MAINTENANCE = 'sdk.EventMaintenance';

    static ASCODE = 'sdk.EventAsCodeEvent';

    static PROJECT_PREFIX = 'sdk.EventProject';
    static PROJECT_DELETE = 'sdk.EventProjectDelete';
    static PROJECT_VARIABLE_PREFIX = 'sdk.EventProjectVariable';
    static PROJECT_PERMISSION_PREFIX = 'sdk.EventProjectPermission';
    static PROJECT_KEY_PREFIX = 'sdk.EventProjectKey';
    static PROJECT_INTEGRATION_PREFIX = 'sdk.EventProjectIntegration';

    static ENVIRONMENT_PREFIX = 'sdk.EventEnvironment';

    static APPLICATION_PREFIX = 'sdk.EventApplication';
    static APPLICATION_ADD = 'sdk.EventApplicationAdd';
    static APPLICATION_UPDATE = 'sdk.EventApplicationUpdate';
    static APPLICATION_DELETE = 'sdk.EventApplicationDelete';

    static PIPELINE_PREFIX = 'sdk.EventPipeline';
    static PIPELINE_ADD = 'sdk.EventPipelineAdd';
    static PIPELINE_UPDATE = 'sdk.EventPipelineUpdate';
    static PIPELINE_DELETE = 'sdk.EventPipelineDelete';
    static PIPELINE_PARAMETER_PREFIX = 'sdk.EventPipelineParameter';

    static WORKFLOW_PREFIX = 'sdk.EventWorkflow';
    static WORKFLOW_ADD = 'sdk.EventWorkflowAdd';
    static WORKFLOW_UPDATE = 'sdk.EventWorkflowUpdate';
    static WORKFLOW_DELETE = 'sdk.EventWorkflowDelete';

    static RUN_WORKFLOW_PREFIX = 'sdk.EventRunWorkflow';
    static RUN_WORKFLOW_JOB = 'sdk.EventRunWorkflowJob';
    static RUN_WORKFLOW_NODE = 'sdk.EventRunWorkflowNode';
    static RUN_WORKFLOW_OUTGOING_HOOK = 'sdk.EventRunWorkflowOutgoingHook';

    static WORKFLOW_RETENTION_DRYRUN = 'sdk.EventRetentionWorkflowDryRun';

    static BROADCAST_PREFIX = 'sdk.EventBroadcast';
    static BROADCAST_ADD = 'sdk.EventBroadcastAdd';
    static BROADCAST_UPDATE = 'sdk.EventBroadcastUpdate';
    static BROADCAST_DELETE = 'sdk.EventBroadcastDelete';

    static ACTION_PREFIX = 'sdk.EventAction';

    static OPERATION = 'sdk.EventOperation';
}
