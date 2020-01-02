export class Variable {
    id: number;
    name: string;
    type: string;
    value: string;
    description: string;

    // flag to know if variable data has changed
    hasChanged: boolean;
    updating: boolean;
    previousName: string;

    constructor() {
        this.name = '';
        this.type = 'string';
        this.value = '';
        this.description = '';
    }
}

export class VariableAudit {
    id: number;
    variable_id: number;
    type: string;
    variable_before: Variable;
    variable_after: Variable;
    versionned: Date;
    author: string;
}
