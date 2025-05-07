export interface WorkflowGenerateResponse {
    readonly error: string;
    readonly workflow: any;
}

export interface WorkflowGenerateRequest {
    filePath: string;
    params: {[key: string]: string};
}
