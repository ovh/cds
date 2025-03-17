export class ProjectWebHook {
    id: string;
    vcs_server: string;
    repository: string;
    workflow: string;
    type: string;
    created: string;
    username: string;
}

export class PostProjectWebHook {
    vcs_server: string;
    repository: string;
    workflow: string;
    type: string;

    constructor() {
        this.type = HookType.Repository;
    }
}

export class PostResponseCreateHook {
    url: string;
    hook_sign_key: string;
}

export enum HookType {
    Repository = "repository",
    //Workflow = "workflow"
}