import { V2Job, V2JobGate, V2WorkflowRunJobEvent } from "./v2.workflow.run.model";
import { V2WorkflowRunJob } from "./v2.workflow.run.model";

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
    type: GraphNodeType;
    name: string;
    depends_on: Array<string>;
    sub_graph: Array<GraphNode>;
    job: V2Job;
    gate: V2JobGate;
    run: V2WorkflowRunJob;
    runs: Array<V2WorkflowRunJob>;
    event: V2WorkflowRunJobEvent;

    static generateMatrixOptions(matrix: { [key: string]: Array<string> }): Array<Map<string, string>> {
        const generateMatrix = (matrix: { [key: string]: string[] }, keys: string[], keyIndex: number, current: Map<string, string>, alls: Array<Map<string, string>>) => {
            if (current.size == keys.length) {
                let combi = new Map<string, string>();
                current.forEach((v, k) => {
                    combi.set(k, v);
                });
                alls.push(combi);
                return;
            }
            let key = keys[keyIndex];
            let values = matrix[key];
            values.forEach(v => {
                current.set(key, v);
                generateMatrix(matrix, keys, keyIndex + 1, current, alls);
                current.delete(key);
            });
        };
        let alls = new Array<Map<string, string>>();
        generateMatrix(matrix, Object.keys(matrix), 0, new Map<string, string>(), alls);
        return alls;
    }
}

export enum GraphNodeType {
    Job = 'job',
    Stage = "stage",
    Matrix = "matrix"
}





