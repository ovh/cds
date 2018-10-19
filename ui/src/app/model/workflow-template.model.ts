import { AuditWorkflowTemplate } from './audit.model';
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
    first_audit: AuditWorkflowTemplate;
    last_audit: AuditWorkflowTemplate;
}

export class WorkflowTemplateParameter {
    key: string;
    type: string;
    required: boolean;
}

export class PipelineTemplate {
    value: string;
}
