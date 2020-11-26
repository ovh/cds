export class WorkflowRetentoinDryRunResponse {
    nb_runs_to_analyze: number;
}
export class RunToKeep {
    id: number;
    status: string;
    num: string;
}

export class RetentionDryRunEvent {
    runs: Array<RunToKeep>;
    nb_runs_analyzed: number;
    status: string;
    error: string;
}
