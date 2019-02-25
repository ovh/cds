import { WorkflowTemplate } from './workflow-template.model';

export class Audit {
    id: number;
    triggered_by: string;
    created: string;
    event_type: string;
}

export class AuditWorkflow extends Audit {
    project_key: string;
    workflow_id: string;
    data_type: string;
    data_before: string;
    data_after: string;
}

export class AuditWorkflowTemplate extends Audit {
    workflow_template_id: string;
    change_message: string;
    data_before: WorkflowTemplate;
    data_after: WorkflowTemplate;
}

export class AuditWorkflowTemplateInstance extends Audit {
    workflow_template_instance_id: string;
    data_type: string;
    data_before: string;
    data_after: string;
}

export class AuditAction extends Audit {
    action_id: string;
    data_type: string;
    data_before: string;
    data_after: string;
}
