import {VCSStrategy} from './vcs.model';

export class PerformAsCodeResponse {
    workflowName: string;
    msgs: any;

    constructor() {
    }
}
export class Operation {
    uuid: string;
    url: string;
    strategy: VCSStrategy;
    vcs_server: string;
    repo_fullname: string;
    repository_info: OperationRepositoryInfo;
    setup: OperationSetup;
    load_files: OperationLoadFiles;
    status: number;
    error: string;

    constructor() {
        this.strategy = new VCSStrategy();
        this.repository_info = new OperationRepositoryInfo();
    }
}

// response from api
export class OperationRepositoryInfo {
    name: string;
    fetch_url: string;
    default_branch: string;
}

// Response from api
export class OperationLoadFiles {
    pattern: string;
    results: {};
}

// from hook
export class OperationSetup {
    checkout: OperationCheckout;
}

// from hook
export class OperationCheckout {
    branch: string;
    commit: string;
}

