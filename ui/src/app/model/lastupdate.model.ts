export class ProjectLastUpdates {
    name: string;
    username: string;
    last_modified: number;
    applications: Array<LastModification>;
    pipelines: Array<LastModification>;
    environments: Array<LastModification>;
}

export class LastModification {
    name: string;
    username: string;
    last_modified: number;
}
