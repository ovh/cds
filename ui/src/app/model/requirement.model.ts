export class Requirement {
    name: string;
    type: string;
    value: string;

    constructor(type: string) {
        this.name = '';
        this.type = type;
        this.value = '';
    }
}
