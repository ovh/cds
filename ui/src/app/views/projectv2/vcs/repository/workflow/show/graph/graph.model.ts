export class StepStatus {
    step_order: number;
    status: string;
    start: string;
    done: string;
}

export class JobRun {
    status: string;
    step_status: Array<StepStatus>;
}

export class GraphNode {
    name: string;
    depends_on: Array<string>;
    sub_graph: Array<GraphNode>;
    run: JobRun;
    type: string;
}

export const GraphNodeTypeJob = "job";
export const GraphNodeTypeStage = "stage";
