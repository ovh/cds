import { WithKey } from '../shared/table/data-table.component';
import { AuditWorkflowTemplate, AuditWorkflowTemplateInstance } from './audit.model';
import { Group } from './group.model';
import { Project } from './project.model';
import { Workflow } from './workflow.model';

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

export class WorkflowTemplateInstance implements WithKey {
    workflow_template_version: number;
    request: WorkflowTemplateRequest;
    project: Project;
    workflow: Workflow;
    first_audit: AuditWorkflowTemplateInstance;
    last_audit: AuditWorkflowTemplateInstance;
    workflow_name: string;

    constructor(wti?: any) {
        if (wti) {
            this.workflow_template_version = wti.workflow_template_version;
            this.request = wti.request;
            this.project = wti.project;
            this.workflow = wti.workflow;
            this.first_audit = wti.first_audit;
            this.last_audit = wti.last_audit;
            this.workflow_name = wti.workflow_name;
        }
    }

    key(): string {
        return this.project.key + '/' + (this.workflow ? this.workflow.name : this.workflow_name);
    }

    status(wt: WorkflowTemplate): InstanceStatus {
        if (!this.workflow) {
            return InstanceStatus.NOT_IMPORTED;
        }
        return this.workflow_template_version === wt.version ? InstanceStatus.UP_TO_DATE : InstanceStatus.NOT_UP_TO_DATE;
    }
}

export enum InstanceStatus {
    NOT_IMPORTED = 'workflow_template_not_imported_yet',
    UP_TO_DATE = 'common_up_to_date',
    NOT_UP_TO_DATE = 'common_not_up_to_date'
}

export class InstanceStatusUtil {
    public static color(status: InstanceStatus): string {
        switch (status) {
            case InstanceStatus.UP_TO_DATE:
                return 'green';
            case InstanceStatus.NOT_UP_TO_DATE:
                return 'red';
            case InstanceStatus.NOT_IMPORTED:
                return 'orange';
        }
        return 'blue';
    }
}
