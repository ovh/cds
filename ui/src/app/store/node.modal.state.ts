import { Action, createSelector, State, StateContext } from '@ngxs/store';
import {Project} from 'app/model/project.model';
import {WNode, Workflow} from 'app/model/workflow.model';
import {
    CleanWorkflowNodeModal,
    OpenWorkflowNodeModal,
    UpdateNodeInModal
} from 'app/store/node.modal.action';

export class NodeModalStateModel {
    public node: WNode;
    public project: Project;
    public workflow: Workflow;
}

export function getInitialNodeModalState(): NodeModalStateModel {
    return {
        node: null,
        workflow: null,
        project: null
    };
}

@State<NodeModalStateModel>({
    name: 'nodemodal',
    defaults: getInitialNodeModalState()
})
export class NodeModalState {

    static getCurrent() {
        return createSelector(
            [NodeModalState],
            (state: NodeModalStateModel): NodeModalStateModel => state
        );
    }

    constructor() {}

    @Action(OpenWorkflowNodeModal)
    open(ctx: StateContext<NodeModalStateModel>, action: OpenWorkflowNodeModal) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            node: action.payload.node,
            project: action.payload.project,
            workflow: action.payload.workflow
        });
    }

    @Action(CleanWorkflowNodeModal)
    close(ctx: StateContext<NodeModalStateModel>) {
        ctx.setState(getInitialNodeModalState());
    }

    @Action(UpdateNodeInModal)
    update(ctx: StateContext<NodeModalStateModel>, action: UpdateNodeInModal) {
        const state = ctx.getState();
        let node = Workflow.getNodeByRef(state.node.ref, action.payload.workflow);
        ctx.setState({
            ...state,
            node: node,
            project: state.project,
            workflow: action.payload.workflow
        });
    }
}
