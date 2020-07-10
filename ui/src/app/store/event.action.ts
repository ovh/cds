import { Event } from '../model/event.model';

export class AddEvent {
    static readonly type = '[Event] Add event';
    constructor(public payload: Event) { }
}
