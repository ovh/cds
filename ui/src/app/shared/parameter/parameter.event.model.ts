import {Parameter} from '../../model/parameter.model';

export class ParameterEvent {
    type: string;
    parameter: Parameter;

    constructor(type: string, p: Parameter) {
        this.type = type;
        this.parameter = p;
    }
}
