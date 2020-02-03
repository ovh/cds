import { AsCodeEventData } from 'app/model/ascode.model';

export class ResyncEvents {
    static readonly type = '[AsCode] Resync Events';
    constructor() { }
}

export class AsCodeEvent {
    static readonly type = '[AsCode] Received event';
    constructor(public payload: {data: AsCodeEventData}) {}
}
