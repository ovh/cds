import { IdName } from './project.model';

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
    warnings: string[];
}

export class WorkflowDependencies {
    pipelines: Array<IdName>;
    applications: Array<IdName>;
    environments: Array<IdName>;

    constructor(obj:  WorkflowDependencies) {
        this.applications = obj.applications;
        this.environments = obj.environments;
        this.pipelines = obj.pipelines;
    }

    isEmpty(): boolean {
        if (!this.pipelines && !this.applications && !this.environments) {
            return true;
        }
        return this.pipelines?.length === 0 && this.applications?.length === 0 && this.environments?.length === 0;
    }
}

export class WorkflowDeletedDependencies {
    deleted_dependencies: WorkflowDependencies;
    unlinked_as_code_dependencies: WorkflowDependencies;
}
