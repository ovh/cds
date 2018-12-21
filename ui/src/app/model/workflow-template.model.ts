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
    environments: Array<EnvironmentTemplate>;
    version: number;
    group: Group;
    first_audit: AuditWorkflowTemplate;
    last_audit: AuditWorkflowTemplate;
    editable: boolean;
    change_message: string;
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

export class EnvironmentTemplate {
    value: string;
}

export class WorkflowTemplateRequest {
    project_key: string
    workflow_name: string
    parameters: { [key: string]: string; }
}

export class WorkflowTemplateApplyResult {
    msgs: Array<string>;
    workflow_name: string;
}

export class WorkflowTemplateInstance {
    workflow_template_version: number;
    request: WorkflowTemplateRequest;
}
