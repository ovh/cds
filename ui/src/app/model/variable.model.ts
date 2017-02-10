export class Variable {
    id: number;
    name: string;
    type: string;
    value: string;
    description: string;

    // flag to know if variable data has changed
    hasChanged: boolean;
    updating: boolean;

    constructor() {
        this.name = '';
        this.type = 'string';
        this.value = '';
        this.description = '';
    }
}
