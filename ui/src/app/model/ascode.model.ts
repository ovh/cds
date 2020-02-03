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

    static FromEventsmanager(data: {}): AsCodeEventData {
        let asCodeEventData = new AsCodeEventData();
        asCodeEventData.workflows = data['Workflows'];
        asCodeEventData.pipelines = data['Pipelines'];
        asCodeEventData.applications = data['Applications'];
        asCodeEventData.environments = data['Environments'];
        return asCodeEventData;
    }
}

export class AsCodeEventDataValue {
    [key: number]: string;
}


