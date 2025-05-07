import { Project } from 'app/model/project.model';

export class SetCurrentProjectV2 {
    static readonly type = '[Event] Set current project V2';
    constructor(public payload: Project) { }
}
