import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
import { Event } from '../model/event.model';
import * as actionEvent from './event.action';

export class EventStateModel {
    public last: Event;
}
@State<EventStateModel>({
    name: 'event',
    defaults: {
        last: null,
    }
})
@Injectable()
export class EventState {
    constructor() { }

    @Selector()
    static last(state: EventStateModel) {
        return state.last
    }

    @Action(actionEvent.AddEvent)
    add(ctx: StateContext<EventStateModel>, action: actionEvent.AddEvent) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            last: action.payload
        });
    }
}
