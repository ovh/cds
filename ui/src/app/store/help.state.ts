import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
import { Help } from '../model/help.model';
import * as actionHelp from './help.action';

export class HelpStateModel {
    public last: Help;
}
@State<HelpStateModel>({
    name: 'help',
    defaults: {
        last: null,
    }
})
@Injectable()
export class HelpState {
    constructor() { }

    @Selector()
    static last(state: HelpStateModel) {
        return state.last
    }

    @Action(actionHelp.AddHelp)
    add(ctx: StateContext<HelpStateModel>, action: actionHelp.AddHelp) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            last: action.payload
        });
    }
}
