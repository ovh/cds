
import { Application } from 'app/model/application.model';
import { GroupPermission } from 'app/model/group.model';
import { ProjectIntegration } from 'app/model/integration.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Label, LoadOpts, Project } from 'app/model/project.model';
import { Variable } from 'app/model/variable.model';
import { Workflow } from 'app/model/workflow.model';

// Use to load fetched Project in our app
export class LoadProject {
    static readonly type = '[Project] Load Project';
    constructor(public payload: Project) { }
}

// Use to fetch Project from backend
export class FetchProject {
    static readonly type = '[Project] Fetch Project';
    constructor(public payload: { projectKey: string, opts: LoadOpts[] }) { }
}

// Use to resync Project from backend
export class ResyncProject {
    static readonly type = '[Project] Resync Project';
    constructor(public payload: { projectKey: string, opts: LoadOpts[] }) { }
}

export class AddProject {
    static readonly type = '[Project] Add Project';
    constructor(public payload: Project) { }
}

export class UpdateProject {
    static readonly type = '[Project] Update Project';
    constructor(public payload: Project) { }
}

export class DeleteProject {
    static readonly type = '[Project] Delete Project';
    constructor(public payload: { projectKey: string }) { }
}

export class ExternalChangeProject {
    static readonly type = '[Project] External Change Project';
    constructor(public payload: { projectKey: string }) { }
}

export class DeleteProjectFromCache {
    static readonly type = '[Project] Delete Project From cache';
    constructor(public payload: { projectKey: string }) { }
}

//  ------- Misc --------- //
export class UpdateFavoriteProject {
    static readonly type = '[Project] Update Project Favorite';
    constructor(public payload: { projectKey: string }) { }
}

//  ------- Variable --------- //
export class ResyncVariablesInProject {
    static readonly type = '[Project] Resync Variables in Project';
    constructor(public payload: { projectKey: string }) { }
}
export class FetchVariablesInProject {
    static readonly type = '[Project] Fetch Variables in Project';
    constructor(public payload: { projectKey: string }) { }
}
export class LoadVariablesInProject {
    static readonly type = '[Project] Load Variables in Project';
    constructor(public payload: Variable[]) { }
}
export class AddVariableInProject {
    static readonly type = '[Project] Add Variable in Project';
    constructor(public payload: Variable) { }
}
export class UpdateVariableInProject {
    static readonly type = '[Project] Update Variable in Project';
    constructor(public payload: { variableName: string, changes: Variable }) { }
}
export class DeleteVariableInProject {
    static readonly type = '[Project] Delete Variable in Project';
    constructor(public payload: Variable) { }
}

//  ------- Application --------- //
export class AddApplicationInProject {
    static readonly type = '[Project] Add application in Project';
    constructor(public payload: Application) { }
}
export class RenameApplicationInProject {
    static readonly type = '[Project] Rename application in Project';
    constructor(public payload: { previousAppName: string, newAppName: string }) { }
}
export class DeleteApplicationInProject {
    static readonly type = '[Project] Delete application in Project';
    constructor(public payload: { applicationName: string }) { }
}

//  ------- Workflow --------- //
export class AddWorkflowInProject {
    static readonly type = '[Project] Add Workflow in Project';
    constructor(public payload: Workflow) { }
}
export class RenameWorkflowInProject {
    static readonly type = '[Project] Rename workflow in Project';
    constructor(public payload: { previousWorkflowName: string, newWorkflowName: string }) { }
}
export class DeleteWorkflowInProject {
    static readonly type = '[Project] Delete Workflow in Project';
    constructor(public payload: { workflowName: string }) { }
}

//  ------- Pipeline --------- //
export class AddPipelineInProject {
    static readonly type = '[Project] Add Pipeline in Project';
    constructor(public payload: Pipeline) { }
}
export class RenamePipelineInProject {
    static readonly type = '[Project] Rename pipeline in Project';
    constructor(public payload: { previousPipName: string, newPipName: string }) { }
}
export class DeletePipelineInProject {
    static readonly type = '[Project] Delete Pipeline in Project';
    constructor(public payload: { pipelineName: string }) { }
}

//  ------- Group Permission --------- //
export class AddGroupInProject {
    static readonly type = '[Project] Add Group in Project';
    constructor(public payload: { projectKey: string, group: GroupPermission }) { }
}
export class UpdateGroupInProject {
    static readonly type = '[Project] Update Group in Project';
    constructor(public payload: { projectKey: string, group: GroupPermission }) { }
}
export class DeleteGroupInProject {
    static readonly type = '[Project] Delete Group in Project';
    constructor(public payload: { projectKey: string, group: GroupPermission }) { }
}

//  ------- Label --------- //
export class AddLabelInProject {
    static readonly type = '[Project] Add Label in Project';
    constructor(public payload: { projectKey: string, label: Label }) { }
}
export class DeleteLabelProject {
    static readonly type = '[Project] Delete Label in Project';
    constructor(public payload: { projectKey: string, label: Label }) { }
}
export class AddLabelWorkflowInProject {
    static readonly type = '[Project] Add Label on Workflow in Project';
    constructor(public payload: { workflowName: string, label: Label }) { }
}
export class DeleteLabelWorkflowInProject {
    static readonly type = '[Project] Delete Label on Workflow in Project';
    constructor(public payload: { workflowName: string, labelId: number }) { }
}

//  ------- Integration --------- //
export class AddIntegrationInProject {
    static readonly type = '[Project] Add Integration in Project';
    constructor(public payload: { projectKey: string, integration: ProjectIntegration }) { }
}
export class DeleteIntegrationProject {
    static readonly type = '[Project] Delete Integration in Project';
    constructor(public payload: { projectKey: string, integration: ProjectIntegration }) { }
}
