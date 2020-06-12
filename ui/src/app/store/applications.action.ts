
import { Application } from 'app/model/application.model';
import { ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { Variable } from 'app/model/variable.model';

// Use to load fetched application in our app
export class LoadApplication {
    static readonly type = '[Application] Load Application';
    constructor(public payload: Application) { }
}

// Use to fetch application from backend
export class FetchApplication {
    static readonly type = '[Application] Fetch Application';
    constructor(public payload: { projectKey: string, applicationName: string }) { }
}

export class AddApplication {
    static readonly type = '[Application] Add Application';
    constructor(public payload: { projectKey: string, application: Application }) { }
}

export class UpdateApplication {
    static readonly type = '[Application] Update Application';
    constructor(public payload: { projectKey: string, applicationName: string, changes: Application }) { }
}

export class DeleteApplication {
    static readonly type = '[Application] Delete Application';
    constructor(public payload: { projectKey: string, applicationName: string }) { }
}

export class FetchApplicationOverview {
    static readonly type = '[Application] Load Application Overview';
    constructor(public payload: { projectKey: string, applicationName: string }) { }
}

export class CloneApplication {
    static readonly type = '[Application] Clone Application';
    constructor(public payload: { projectKey: string, newApplication: Application, clonedAppName: string }) { }
}

//  ------- Variables ---------
export class AddApplicationVariable {
    static readonly type = '[Application] Add Application Variable';
    constructor(public payload: { projectKey: string, applicationName: string, variable: Variable }) { }
}

export class UpdateApplicationVariable {
    static readonly type = '[Application] Update Application Variable';
    constructor(public payload: { projectKey: string, applicationName: string, variableName: string, variable: Variable }) { }
}

export class DeleteApplicationVariable {
    static readonly type = '[Application] Delete Application Variable';
    constructor(public payload: { projectKey: string, applicationName: string, variable: Variable }) { }
}


//  ------- Keys --------- //
export class AddApplicationKey {
    static readonly type = '[Application] Add Application Key';
    constructor(public payload: { projectKey: string, applicationName: string, key: Key }) { }
}

export class DeleteApplicationKey {
    static readonly type = '[Application] Delete Application Key';
    constructor(public payload: { projectKey: string, applicationName: string, key: Key }) { }
}

//  ------- Deployment strategies --------- //

export class AddApplicationDeployment {
    static readonly type = '[Application] Add Application Deployment';
    constructor(public payload: { projectKey: string, applicationName: string, integration: ProjectIntegration }) { }
}

export class UpdateApplicationDeployment {
    static readonly type = '[Application] Update Application Deployment';
    constructor(public payload: { projectKey: string, applicationName: string, deploymentName: string, config: Map<string, any> }) { }
}

export class DeleteApplicationDeployment {
    static readonly type = '[Application] Delete Application Deployment';
    constructor(public payload: { projectKey: string, applicationName: string, integrationName: string }) { }
}

//  ------- VCS --------- //
export class ConnectVcsRepoOnApplication {
    static readonly type = '[Application] Connect a VCS repository on Application';
    constructor(public payload: { projectKey: string, applicationName: string, repoManager: string, repoFullName: string }) { }
}

export class DeleteVcsRepoOnApplication {
    static readonly type = '[Application] Delete a VCS repository on Application';
    constructor(public payload: { projectKey: string, applicationName: string, repoManager: string }) { }
}

//  ------- Misc --------- //
export class ExternalChangeApplication {
    static readonly type = '[Application] External Change Application';
    constructor(public payload: { projectKey: string, applicationName: string }) { }
}

export class ResyncApplication {
    static readonly type = '[Application] Resync Application';
    constructor(public payload: { projectKey: string, applicationName: string }) { }
}

export class ClearCacheApplication {
    static readonly type = '[Application] Clear cache Application';
    constructor() { }
}

export class CancelApplicationEdition {
    static readonly type = '[Application] Cancel application edition';
    constructor() { }
}


