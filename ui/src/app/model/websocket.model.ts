import { Event } from 'app/model/event.model';

export enum WebsocketFilterType {
    PROJECT = 'project',
    WORKFLOW = 'workflow',
    WORKFLOW_RUN = 'workflow-run',
    WORKFLOW_NODE_RUN = 'workflow-node-run',
    PIPELINE = 'pipeline',
    APPLICATION = 'application',
    ENVIRONMENT = 'environment',
    QUEUE = 'queue',
    OPERATION = 'operation'
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
