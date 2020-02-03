
import { Job } from 'app/model/job.model';
import { Parameter } from 'app/model/parameter.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Stage } from 'app/model/stage.model';

// Use to load fetched Pipeline in our app
export class LoadPipeline {
    static readonly type = '[Pipeline] Load Pipeline';
    constructor(public payload: { projectKey: string, pipeline: Pipeline }) { }
}

// Use to fetch Pipeline from backend
export class FetchPipeline {
    static readonly type = '[Pipeline] Fetch Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string }) { }
}

export class AddPipeline {
    static readonly type = '[Pipeline] Add Pipeline';
    constructor(public payload: { projectKey: string, pipeline: Pipeline }) { }
}

export class ImportPipeline {
    static readonly type = '[Pipeline] Import Pipeline';
    constructor(public payload: { projectKey: string, pipelineCode: string, pipName?: string, force?: boolean }) { }
}

export class UpdatePipeline {
    static readonly type = '[Pipeline] Update Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string, changes: Pipeline }) { }
}

export class DeletePipeline {
    static readonly type = '[Pipeline] Delete Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string }) { }
}

//  ------- Audit ---------
export class FetchPipelineAudits {
    static readonly type = '[Pipeline] Fetch Pipeline Audits';
    constructor(public payload: { projectKey: string, pipelineName: string }) { }
}

export class RollbackPipeline {
    static readonly type = '[Pipeline] Rollback Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string, auditId: number }) { }
}

//  ------- Parameter ---------
export class AddPipelineParameter {
    static readonly type = '[Pipeline] Add Pipeline Parameter';
    constructor(public payload: { projectKey: string, pipelineName: string, parameter: Parameter }) { }
}

export class UpdatePipelineParameter {
    static readonly type = '[Pipeline] Update Pipeline Parameter';
    constructor(public payload: { projectKey: string, pipelineName: string, parameterName: string, parameter: Parameter }) { }
}

export class DeletePipelineParameter {
    static readonly type = '[Pipeline] Delete Pipeline Parameter';
    constructor(public payload: { projectKey: string, pipelineName: string, parameter: Parameter }) { }
}

//  ------- Workflow --------- //
export class AddPipelineStage {
    static readonly type = '[Pipeline] Add Pipeline Stage';
    constructor(public payload: { projectKey: string, pipelineName: string, stage: Stage }) { }
}

export class MovePipelineStage {
    static readonly type = '[Pipeline] Move Pipeline Stage';
    constructor(public payload: { projectKey: string, pipeline: Pipeline, stage: Stage }) { }
}

export class UpdatePipelineStage {
    static readonly type = '[Pipeline] Update Pipeline Stage';
    constructor(public payload: { projectKey: string, pipelineName: string, changes: Stage }) { }
}

export class DeletePipelineStage {
    static readonly type = '[Pipeline] Delete Pipeline Stage';
    constructor(public payload: { projectKey: string, pipelineName: string, stage: Stage }) { }
}

export class AddPipelineJob {
    static readonly type = '[Pipeline] Add Pipeline Job';
    constructor(public payload: { projectKey: string, pipelineName: string, stage: Stage, job: Job }) { }
}

export class UpdatePipelineJob {
    static readonly type = '[Pipeline] Update Pipeline Job';
    constructor(public payload: { projectKey: string, pipelineName: string, stage: Stage, changes: Job }) { }
}

export class DeletePipelineJob {
    static readonly type = '[Pipeline] Delete Pipeline Job';
    constructor(public payload: { projectKey: string, pipelineName: string, stage: Stage, job: Job }) { }
}

//  ------- Misc --------- //
export class FetchAsCodePipeline {
    static readonly type = '[Pipeline] Fetch Pipeline As Code';
    constructor(public payload: { projectKey: string, pipelineName: string }) { }
}

export class PreviewPipeline {
    static readonly type = '[Pipeline] Preview Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string, pipCode: string }) { }
}

export class ExternalChangePipeline {
    static readonly type = '[Pipeline] External Change Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string }) { }
}

export class ResyncPipeline {
    static readonly type = '[Pipeline] Resync Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string }) { }
}

export class DeleteFromCachePipeline {
    static readonly type = '[Pipeline] Delete from cache Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string }) { }
}

export class ClearCachePipeline {
    static readonly type = '[Pipeline] Clear cache Pipeline';
    constructor() { }
}

export class CancelPipelineEdition {
    static readonly type = '[Pipeline] Cancel pipeline edition';
    constructor() { }
}


