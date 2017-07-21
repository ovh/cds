export class ProjectLastUpdates {
    name: string;
    username: string;
    last_modified: number;
    applications: Array<LastModification>;
    pipelines: Array<LastModification>;
    environments: Array<LastModification>;
}

export class LastModification {
    key: string;
    name: string;
    username: string;
    last_modified: number;
    type: string;
}
