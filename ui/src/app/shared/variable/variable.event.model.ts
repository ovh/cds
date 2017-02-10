import {Variable} from '../../model/variable.model';

export class VariableEvent {
    type: string;
    variable: Variable;

    constructor(type: string, v: Variable) {
        this.type = type;
        this.variable = v;
    }
}
