export enum EntityType {
    WorkerModel = 'WorkerModel',
    Action = 'Action',
    Workflow = 'Workflow',
    Job = 'Job'
}

export class Entity {
    id: string;
    name: string;
    type: EntityType;
    branch: string;
    data: string;
    file_path: string;
}

export class EntityFullName {
    name: string;
    branch: string;
    vcs_name: string;
    repo_name: string;
    project_key: string;
}

export class EntityCheckResponse {
    messages: string[];
}
