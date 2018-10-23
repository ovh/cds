import { AuditWorkflowTemplate } from './audit.model';
import { Group } from './group.model';

export class WorkflowTemplate {
    id: number;
    group_id: number;
    name: string;
    slug: string;
    description: string;
    parameters: Array<WorkflowTemplateParameter>;
    value: string;
    pipelines: Array<PipelineTemplate>;
    applications: Array<ApplicationTemplate>;
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

export class ApplicationTemplate {
    value: string;
}
