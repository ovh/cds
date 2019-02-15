
import { LoadOpts, Project } from 'app/model/project.model';

// Use to load fetched Project in our app
export class LoadProject {
    static readonly type = '[Project] Load Project';
    constructor(public payload: Project) { }
}

// Use to fetch Project from backend
export class FetchProject {
    static readonly type = '[Project] Fetch Project';
    constructor(public payload: { projectKey: string, opts: LoadOpts[] }) { }
}

export class AddProject {
    static readonly type = '[Project] Add TodoItem';
    constructor(public payload: Project) { }
}

export class UpdateProject {
    static readonly type = '[Project] Update Project';
    constructor(public payload: { projectKey: string, changes: Project }) { }
}

export class DeleteProject {
    static readonly type = '[Project] Delete Project';
    constructor(public payload: { projectKey: string }) { }
}

export class ExternalChangeProject {
    static readonly type = '[Project] External Change Project';
    constructor(public payload: { projectKey: string }) { }
}
