import { FullEventV2 } from "./event-v2.model";

export enum WebsocketV2FilterType {
    GLOBAL = 'global',
    PROJECT = 'project',
    PROJECT_RUNS = 'project-runs'
}

export class WebsocketV2Filter {
    type: WebsocketV2FilterType;
    project_key: string;
    project_runs_params: string;
}

export class WebsocketV2Event {
    status: string;
    error: string;
    event: FullEventV2;
}
