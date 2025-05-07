import { FullEventV2 } from 'app/model/event-v2.model';

export class AddEventV2 {
    static readonly type = '[Event] Add event V2';
    constructor(public payload: FullEventV2) { }
}
