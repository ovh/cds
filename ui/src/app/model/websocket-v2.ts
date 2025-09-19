import { FullEventV2 } from "./event-v2.model";

export enum WebsocketV2FilterType {
    GLOBAL = 'global',
    PROJECT = 'project',
    PROJECT_RUNS = 'project-runs',
    PROJECT_PURGE_REPORT = 'project-purge-report',
    QUEUE = 'queue'
}

export class WebsocketV2Filter {
    type: WebsocketV2FilterType;
    project_key: string;
    project_runs_params: string;
    purge_report_id: string;
}

export class WebsocketV2Event {
    status: string;
    error: string;
    event: FullEventV2;
}
