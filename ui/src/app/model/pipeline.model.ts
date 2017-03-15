import {Parameter} from './parameter.model';
import {Stage} from './stage.model';
import {GroupPermission} from './group.model';
import {User} from './user.model';
import {Application} from './application.model';
import {Environment} from './environment.model';
import {Artifact} from './artifact.model';
import {Job} from './job.model';

export class Pipeline {
    id: number;
    name: string;
    type: string;
    last_pipeline_build: PipelineBuild;
    stages: Array<Stage>;
    groups: Array<GroupPermission>;
    parameters: Array<Parameter>;
    permission: number;
    last_modified: number;
    projectKey: string;
    attached_application: Array<Application>;

    // true if someone has updated the pipeline ( used for warnings )
    externalChange: boolean;

    public static hasParameterWithoutValue(pipeline: Pipeline) {
        if (pipeline.parameters) {
            let emptyParams = pipeline.parameters.filter(p => {
                return !p.value || p.value === '';
            });
            return emptyParams.length > 0;
        }
        return false;
    }

    /**
     * Merge parameters
     * @param ref
     * @param current
     */
    public static mergeParams(ref: Array<Parameter>, current: Array<Parameter>): Array<Parameter> {
        let params = new Array<Parameter>();
        if (ref) {
            if (!current || current.length === 0) {
                params.push(...ref);
            } else {
                ref.forEach( p => {
                    let found = false;
                    for (let i = 0; i < current.length; i++) {
                        if (current[i].name === p.name) {
                            found = true;
                            params.push(current[i]);
                            break;
                        }
                    }
                    if (!found) {
                        params.push(p);
                    }
                });
            }
        }
        return params;
    }

    constructor() {
        this.attached_application = new Array<Application>();
    }
}

export class PipelineRunRequest {
    parameters: Array<Parameter>;
    env: Environment;
    parent_build_number: number;
    parent_pipeline_id: number;
    parent_environment_id: number;
    parent_application_id: number;

    constructor() {
        this.parameters = new Array<Parameter>();
    }
}

export class PipelineBuild {
    id: number;
    build_number: number;
    version: number;
    parameters: Array<Parameter>;
    status: string;
    start: string;
    done: string;
    stages: Array<Stage>;
    pipeline: Pipeline;
    application: Application;
    environment: Environment;
    trigger: PipelineBuildTrigger;
    artifacts: Array<Artifact>;
    tests: Tests;

    public static GetTriggerSource(pb: PipelineBuild): string {
        if (pb.trigger.scheduled_trigger) {
            return 'CDS scheduler';
        }
        if (pb.trigger.triggered_by && pb.trigger.triggered_by.username && pb.trigger.triggered_by.username !== '') {
            return pb.trigger.triggered_by.username;
        }
        if (pb.trigger.vcs_author) {
            return pb.trigger.vcs_author;
        }
        return '';
    }
}



export class PipelineBuildJob {
    id: number;
    job: Job;
    parameters: Array<Parameter>;
    status: string;
    queued: number;
    start: number;
    done: number;
    model: number;
    pipeline_build_id: number;
}


export class BuildResult {
    status: string;
    step_logs: Log;
}

export interface Log {
    id: number;
    action_build_id: number;
    pipeline_build_id: number;
    timestamp: number;
    step_order: number;
    value: string;
    start: number;
    last_modified: number;
    done: number;
}

export class PipelineBuildTrigger {
    manual_trigger: boolean;
    scheduled_trigger: boolean;
    triggered_by: User;
    parent_pipeline_build: PipelineBuild;
    vcs_branch: string;
    vcs_hash: string;
    vcs_author: string;
}

export enum PipelineType {
    build,
    testing,
    deployment
}

export class Tests {
    pipeline_build_id: number;
    total: number;
    ok: number;
    ko: number;
    skipped: number;
    test_suites: Array<TestSuite>;
}

export class TestSuite {
    disabled: number;
    errors: number;
    failures: number;
    id: string;
    name: string;
    package: string;
    skipped: number;
    total: number;
    time: string;
    timestamp: string;
    tests: Array<TestCase>;
}

export class TestCase {
    classname: string;
    name: string;
    time: string;
    errors: Array<Failure>;
    failures: Array<Failure>;
    status: string;
    skipped: number;
    systemout: InnerResult;
    systemerr: InnerResult;

    // UI param
    displayed: boolean;
}

// Failure contains data related to a failed test.
export class Failure {
    value: string;
    type: string;
    message: string;
}

// InnerResult is used by TestCase
export interface InnerResult {
    value: string;
}
