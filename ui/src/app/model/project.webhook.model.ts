export class ProjectWebHook {
    id: string;
    vcs_server: string;
    repository: string;
    created: string;
    username: string;
}

export class PostProjectRepositoryHook {
    vcs_server: string;
    repository: string;
}

export class PostResponseCreateHook {
    url: string;
    hook_sign_key: string;
}