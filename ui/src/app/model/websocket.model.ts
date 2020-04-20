import { Event } from 'app/model/event.model';

export class WebSocketMessage {
    project_key: string;
    application_name: string;
    pipeline_name: string;
    environment_name: string;
    workflow_name: string;
    workflow_run_num: number;
    workflow_node_run_id: number;
    favorites: boolean;
    queue: boolean;
    operation: string;
    type: string;
}

export class WebSocketEvent {
    status: string;
    error: string;
    event: Event;
}
