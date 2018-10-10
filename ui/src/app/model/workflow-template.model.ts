import { Group } from './group.model';

export class WorkflowTemplate {
    id: number;
    group_id: number;
    name: string;
    description: string;
    parameters: Array<WorkflowTemplateParameter>;
    value: string;
    pipelines: Array<PipelineTemplate>;
    version: number;
    group: Group;
}

export class WorkflowTemplateParameter {
    key: string;
    type: string;
    required: boolean;
}

export class PipelineTemplate {
    value: string;
}
