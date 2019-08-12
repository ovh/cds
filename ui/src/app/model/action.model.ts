import { AuditAction } from './audit.model';
import { Group } from './group.model';
import { Parameter } from './parameter.model';
import { Requirement } from './requirement.model';

export class Action {
    id: number;
    group_id: number;
    name: string;
    step_name: string;
    type: string;
    description = '';
    requirements: Array<Requirement>;
    parameters: Array<Parameter>;
    actions: Array<Action>;
    optional: boolean;
    always_executed: boolean;
    enabled: boolean;
    deprecated: boolean;
    group: Group;
    first_audit: AuditAction;
    last_audit: AuditAction;
    editable: boolean;

    // UI parameter
    hasChanged: boolean;
    loading: boolean;
    showAddStep: boolean;
}

export class Usage {
    pipelines: Array<UsagePipeline>;
    actions: Array<UsageAction>;
}

export class UsagePipeline {
    project_id: number;
    project_key: string;
    project_name: string;
    pipeline_id: number;
    pipeline_name: string;
    action_id: number;
    action_name: string;
    warning: boolean;
}

export class UsageAction {
    group_id: number;
    group_name: string;
    parent_action_name: string;
    action_id: number;
    warning: boolean;
}

export class ActionWarning {
    type: string;
    action: Action;
}
