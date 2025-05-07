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

    constructor() {
        this.name = '';
        this.type = 'string';
        this.value = '';
    }
}