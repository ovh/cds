export class Parameter {
    id: number;
    name: string;
    type: string;
    value: string;
    description: string;

    // flag to know if variable data has changed
    hasChanged: boolean;
    updating: boolean;

    static formatForAPI(parameters: Array<Parameter>): Array<Parameter> {
        return parameters.map(p => {
            let pa = new Parameter();
            pa.name = p.name;
            pa.id = p.id;
            pa.type = p.type;
            pa.description = p.description;
            pa.value = p.value.toString();
           return pa;
        });
    }
}
