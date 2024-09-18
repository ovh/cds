import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
import * as actionEventV2 from './event-v2.action';
import { FullEventV2 } from 'app/model/event-v2.model';

export class EventV2StateModel {
    public last: FullEventV2;
}
@State<EventV2StateModel>({
    name: 'eventv2',
    defaults: {
        last: null,
    }
})
@Injectable()
export class EventV2State {
    constructor() { }

    @Selector()
    static last(state: EventV2StateModel) {
        return state.last;
    }

    @Action(actionEventV2.AddEventV2)
    add(ctx: StateContext<EventV2StateModel>, action: actionEventV2.AddEventV2) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            last: action.payload
        });
    }
}
