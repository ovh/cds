import {WorkflowNodeHook} from '../../../../model/workflow.model';

export class HookEvent {
    type: string; // add update delete
    hook: WorkflowNodeHook;

    constructor(t: string, h: WorkflowNodeHook) {
        this.type = t;
        this.hook = h;
    }
}
