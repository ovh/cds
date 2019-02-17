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
    import_url: string;

    // UI parameter
    hasChanged: boolean;
    loading: boolean;
    showAddStep: boolean;
}

export class Usage {
    Pipelines: Array<UsagePipeline>;
    Actions: Array<UsageAction>;
}

export class UsagePipeline {
    ProjectID: number;
    ProjectKey: string;
    ProjectName: string;
    PipelineID: number;
    PipelineName: string;
    StageID: number;
    StageName: string;
    JobID: number;
    JobName: string;
    ActionID: number;
    ActionName: string;
    Warning: boolean;
}

export class UsageAction {
    ParentActionID: number;
    ParentActionGroupName: string;
    ParentActionName: string;
    ActionID: number;
    ActionName: string;
    Warning: boolean;
}

export class ActionWarning {
    type: string;
    action: Action;
}
