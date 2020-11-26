import { Event } from 'app/model/event.model';

export enum WebsocketFilterType {
    GLOBAL = 'global',
    PROJECT = 'project',
    WORKFLOW = 'workflow',
    WORKFLOW_RUN = 'workflow-run',
    WORKFLOW_NODE_RUN = 'workflow-node-run',
    WORKFLOW_RETENTION_DRYRUN = 'workflow-retention-dryrun',
    PIPELINE = 'pipeline',
    APPLICATION = 'application',
    ENVIRONMENT = 'environment',
    QUEUE = 'queue',
    OPERATION = 'operation',
    TIMELINE = 'timeline',
    ASCODE_EVENT = 'ascode-event'

}

export class WebsocketFilter {
    type: string;
    project_key: string;
    application_name: string;
    pipeline_name: string;
    environment_name: string;
    workflow_name: string;
    workflow_run_num: number;
    workflow_node_run_id: number;
    operation_uuid: string;
}

export class WebsocketEvent {
    status: string;
    error: string;
    event: Event;
}
