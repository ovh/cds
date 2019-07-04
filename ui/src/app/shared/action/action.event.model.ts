import { Action } from 'app/model/action.model';

export class ActionEvent {
    type: string;
    action: Action;

    constructor(type: string, a: Action) {
        this.type = type;
        this.action = a;
    }
}
