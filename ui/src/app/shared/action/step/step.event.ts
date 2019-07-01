import { Action } from 'app/model/action.model';

export class StepEvent {
    type: string;
    step: Action;

    constructor(t: string, s: Action) {
        this.type = t;
        this.step = s;
    }
}
