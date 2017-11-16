export class Requirement {
    name: string;
    type: string;
    value: string;
    opts: string;

    constructor(type: string) {
        this.name = '';
        this.type = type;
        this.value = '';
        this.opts = '';
    }
}
