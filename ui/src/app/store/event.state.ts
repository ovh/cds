import { Injectable } from '@angular/core';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { Event, EventType } from '../model/event.model';
import * as actionEvent from './event.action';

export class EventStateModel {
    public all: Array<Event>;
}
@State<EventStateModel>({
    name: 'event',
    defaults: {
        all: []
    }
})
@Injectable()
export class EventState {
    constructor() { }

    static lastByType(type: EventType) {
        return createSelector([EventState], (state: EventStateModel) => {
            return state.all.filter(e => e.type_event === type);
        });
    }

    @Action(actionEvent.AddEvent)
    add(ctx: StateContext<EventStateModel>, action: actionEvent.AddEvent) {
        const state = ctx.getState();
        // Set a limit to keep only a set of last received events
        if (state.all.length >= 20) {
            ctx.setState({
                ...state,
                all: state.all.slice(1).concat(action.payload)
            });
        } else {
            ctx.setState({
                ...state,
                all: state.all.concat(action.payload)
            });
        }
    }
}
