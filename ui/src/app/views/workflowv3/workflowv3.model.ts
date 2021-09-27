import { SpawnInfo } from 'app/model/pipeline.model';

export class WorkflowV3ValidationResponse {
    valid: boolean;
    error: string;
    workflow: WorkflowV3;
    external_dependencies: Array<any>;
}

export class WorkflowV3 {
    name: string;
    stages: { [name: string]: Stage };
    jobs: { [name: string]: Job };
}

export class WorkflowRunV3 {
    status: string;
    number: number;
    workflow: WorkflowV3;
    job_runs: { [name: string]: Array<JobRun> };
    infos: Array<SpawnInfo>;
}

export class JobRun {
    status: string;
    sub_number: number;
    step_status: Array<StepStatus>;
    // Info from workflow v2 model
    workflow_node_run_id: number;
    workflow_node_job_run_id: number;
}

export class StepStatus {
    step_order: number;
    status: string;
    start: string;
    done: string;
}

export class Stage {
    depends_on: Array<string>;
    conditions: any;
}

export class Job {
    enabled: boolean;
    description: string;
    conditions: any;
    context: any;
    stage: string;
    steps: Array<any>;
    requirements: Array<any>;
    depends_on: Array<string>;
}

export class GraphNode {
    name: string;
    depends_on: Array<string>;
    sub_graph: Array<GraphNode>;
    run: JobRun;
}
