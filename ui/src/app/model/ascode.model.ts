export class AsCodeEvents {
    id: number;
    pullrequest_id: number;
    pullrequest_url: string;
    username: string;
    creation_date: string;
    from_repository: string;
    data: AsCodeEventData;
}

export class AsCodeEventData {
    workflows: AsCodeEventDataValue;
    pipelines: AsCodeEventDataValue;
    applications: AsCodeEventDataValue;
    environments: AsCodeEventDataValue;
}

export class AsCodeEventDataValue {
    [key: number]: string;
}
