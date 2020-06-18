import { Environment } from 'app/model/environment.model';
import { Key } from 'app/model/keys.model';
import { Variable } from 'app/model/variable.model';


export class AddEnvironment {
    static readonly type = '[Environment] Add Environment';
    constructor(public payload: { projectKey: string, environment: Environment }) { }
}

export class CloneEnvironment {
    static readonly type = '[Environment] Clone Environment';
    constructor(public payload: { projectKey: string, cloneName: string, environment: Environment }) { }
}

export class UpdateEnvironment {
    static readonly type = '[Environment] Update environment';
    constructor(public payload: { projectKey: string, environmentName: string, changes: Environment }) { }
}
export class DeleteEnvironment {
    static readonly type = '[Environment] Delete Environment';
    constructor(public payload: { projectKey: string, environment: Environment }) { }
}

// LOAD

export class FetchEnvironment {
    static readonly type = '[Environment] Fetch Single Environment';
    constructor(public payload: { projectKey: string, envName: string }) { }
}

export class LoadEnvironment {
    static readonly type = '[Environment] Load Environment';
    constructor(public payload: {projectKey: string, env: Environment}) { }
}

export class ResyncEnvironment {
    static readonly type = '[Environment] Resync Single Environment';
    constructor(public payload: { projectKey: string, envName: string }) { }
}


// VARIABLE

export class AddEnvironmentVariable {
    static readonly type = '[Environment] Add Environment Variable';
    constructor(public payload: { projectKey: string, environmentName: string, variable: Variable }) { }
}
export class UpdateEnvironmentVariable {
    static readonly type = '[Environment] Update environment variable';
    constructor(public payload: { projectKey: string, environmentName: string, variableName: string, changes: Variable }) { }
}
export class DeleteEnvironmentVariable {
    static readonly type = '[Environment] Delete Environment Variable ';
    constructor(public payload: { projectKey: string, environmentName: string, variable: Variable }) { }
}

// KEY

export class AddEnvironmentKey {
    static readonly type = '[Environment] Add Environment Key';
    constructor(public payload: { projectKey: string, envName: string, key: Key }) { }
}

export class DeleteEnvironmentKey {
    static readonly type = '[Environment] Delete Environment Key';
    constructor(public payload: { projectKey: string, envName: string, key: Key }) { }
}

// Clean
export class CleanEnvironmentState {
    static readonly type = '[Environment] Clean state';
    constructor() { }
}



