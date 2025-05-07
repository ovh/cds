export enum EntityType {
    WorkerModel = 'WorkerModel',
    WorkflowTemplate = 'WorkflowTemplate',
    Action = 'Action',
    Workflow = 'Workflow',
    Job = 'Job'
}

export class EntityTypeUtil {
    public static toURLParam(t: EntityType): string {
        return t.toLowerCase();
    }

    public static fromURLParam(p: string): EntityType {
        for (const [key, value] of Object.entries(EntityType)) {
            if (value.toLowerCase() === p) {
                return value;
            }
        }
        throw `Given param ${p} is not matching any entity type`;
    }
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
    ref: string;
    vcs_name: string;
    repo_name: string;
    project_key: string;
}

export class EntityCheckResponse {
    messages: string[];
}
