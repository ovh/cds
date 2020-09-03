export interface Pipeline {
    id: number;
    name: string;
    description: string;
    icon: string;
    // stages: Array<Stage>;
    // parameters: Array<Parameter>;
    // permission: number;
    // last_modified: number;
    // projectKey: string;
    // usage: Usage;
    // audits: Array<PipelineAudit>;
    // preview: Pipeline;
    asCode: string;
    from_repository: string;

    // true if someone has updated the pipeline ( used for warnings )
    externalChange: boolean;

    // UI Params
    forceRefresh: boolean;
    previewMode: boolean;

}
