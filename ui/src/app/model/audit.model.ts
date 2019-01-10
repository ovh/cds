export class Audit {
    id: number;
    triggered_by: string;
    created: string;
    data_before: string;
    data_after: string;
    event_type: string;
    data_type: string;
}

export class AuditWorkflow extends Audit {
    project_key: string;
    workflow_id: string;
}

export class AuditWorkflowTemplate extends Audit {
    workflow_template_id: string;
    change_message: string;
}
