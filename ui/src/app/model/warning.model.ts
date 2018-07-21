
export class Warning {
    id: number;
    key: string;
    application_name: string;
    pipeline_name: string;
    workflow_name: string;
    environment_name: string;
    type: string;
    element: string;
    hash: string;
    created: string;
    message_params: MessageParams;
    message: string;
    ignored: boolean;

    // UI specific param
    loading: boolean;
}

export interface MessageParams {
    [key: string]: string;
}
