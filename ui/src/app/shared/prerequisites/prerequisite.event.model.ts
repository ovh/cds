import { Prerequisite } from 'app/model/prerequisite.model';

export class PrerequisiteEvent {
    type: string;
    prerequisite: Prerequisite;

    constructor(type: string, p: Prerequisite) {
        this.type = type;
        this.prerequisite = p;
    }
}
