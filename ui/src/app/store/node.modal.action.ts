
import {Project} from 'app/model/project.model';
import {WNode, WNodeHook, Workflow} from 'app/model/workflow.model';

export class OpenWorkflowNodeModal {
    static readonly type = '[ModalNode] OpenNodeModal';
    constructor(public payload: { project: Project, workflow: Workflow, node: WNode, hook: WNodeHook }) { }
}

export class CleanWorkflowNodeModal {
    static readonly type = '[ModalNode] CloseNodeModal';
    constructor(public payload: {}) { }
}

export class UpdateNodeInModal {
    static readonly type = '[ModalNode] UpdateNodeModal';
    constructor(public payload: { workflow: Workflow }) {}
}

