import { Requirement } from 'app/model/requirement.model';

export class RequirementEvent {
    type: string;
    requirement: Requirement;

    constructor(type: string, r: Requirement) {
        this.type = type;
        this.requirement = r;
    }
}
