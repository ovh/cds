
import { Parameter } from 'app/model/parameter.model';
import { Pipeline } from 'app/model/pipeline.model';

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

export class UpdatePipeline {
    static readonly type = '[Pipeline] Update Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string, changes: Pipeline }) { }
}

export class DeletePipeline {
    static readonly type = '[Pipeline] Delete Pipeline';
    constructor(public payload: { projectKey: string, pipelineName: string }) { }
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

//  ------- Misc --------- //
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


