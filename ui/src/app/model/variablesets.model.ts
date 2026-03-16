export class VariableSet {
    id: string;
    name: string;
    items: VariableSetItem[];
   
    constructor() {
        this.name = '';
    }
}

export class VariableSetItem {
    name: string
    type: string
    value: string

    // UI field to know if the value has been modified
    changed: boolean

    constructor() {
        this.name = '';
        this.type = 'string';
        this.value = '';
    }
}