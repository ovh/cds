export const EntityWorkerModel = "WorkerModel";
export const EntityAction = "Action";
export const EntityWorkflow = "Workflow";

export class Entity {
    id: string;
    name: string;
    type: string;
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
