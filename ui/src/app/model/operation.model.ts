import {VCSStrategy} from './vcs.model';

export class PerformAsCodeResponse {
    workflowName: string;
    msgs: any;

    constructor() {
    }
}
export class OperationErrorDetails {
    id: number;
    status: number;
    message: string;
    from: string;

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
    error_details: OperationErrorDetails;

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
    push: OperationPush;
}

// from hook
export class OperationCheckout {
    branch: string;
    commit: string;
}

export class OperationPush {
    from_branch: string;
    to_branch: string;
    message: string;
    pr_link: string;
}

