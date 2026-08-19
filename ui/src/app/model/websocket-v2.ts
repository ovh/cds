import { FullEventV2 } from "./event-v2.model";

export enum WebsocketV2FilterType {
    GLOBAL = 'global',
    PROJECT = 'project',
    PROJECT_RUNS = 'project-runs',
    // Every event of a single workflow run: the run itself, its jobs, their steps and their results.
    PROJECT_RUN = 'project-run',
    PROJECT_PURGE_REPORT = 'project-purge-report',
    QUEUE = 'queue'
}

export class WebsocketV2Filter {
    type: WebsocketV2FilterType;
    project_key: string;
    project_runs_params: string;
    workflow_run_id: string;
    purge_report_id: string;
}

export class WebsocketV2Event {
    status: string;
    error: string;
    event: FullEventV2;
}
