import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { SetCurrentProjectV2 } from 'app/store/project-v2.action';

export class ProjectV2StateModel {
    public current: Project;
}

@State<ProjectV2StateModel>({
    name: 'projectv2',
    defaults: {
        current: null
    }
})
@Injectable()
export class ProjectV2State {

    constructor() { }

    @Selector()
    static current(state: ProjectV2StateModel) {
        return state.current;
    }

    @Action(SetCurrentProjectV2)
    setCurrent(ctx: StateContext<ProjectV2StateModel>, action: SetCurrentProjectV2) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            current: action.payload
        });
    }

}
