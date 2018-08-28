export class Parameter {
    id: number;
    name: string;
    type: string;
    value: string;
    description: string;
    advanced: boolean;

    // flag to know if variable data has changed
    hasChanged: boolean;
    updating: boolean;
    previousName: string;

    // Useful for list on UI
    ref: Parameter;

    static formatForAPI(parameters: Array<Parameter>): Array<Parameter> {
        if (parameters) {
            return parameters.map(this.format);
        }
        return new Array<Parameter>();
    }

    static format(parameter: Parameter): Parameter {
        if (parameter) {
            let pa = new Parameter();
            pa.name = parameter.name;
            pa.id = parameter.id;
            pa.type = parameter.type;
            pa.description = parameter.description;
            pa.value = parameter.value.toString();
            pa.advanced = parameter.advanced;
            return pa;
        }
        return parameter;
    }
}
