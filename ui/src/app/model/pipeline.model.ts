import {ActionWarning} from './action.model';
import {Application} from './application.model';
import {Artifact} from './artifact.model';
import {Environment} from './environment.model';
import {GroupPermission} from './group.model';
import {Job} from './job.model';
import {Parameter} from './parameter.model';
import {Commit} from './repositories.model';
import {Stage} from './stage.model';
import {Usage} from './usage.model';
import {User} from './user.model';

export const pipelineNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');

export class PipelineStatus {
    static BUILDING = 'Building';
    static FAIL = 'Fail';
    static SUCCESS = 'Success';
    static WAITING = 'Waiting';
    static DISABLED = 'Disabled';
    static SKIPPED = 'Skipped';
    static NEVER_BUILT = 'Never Built';
    static STOPPED = 'Stopped';

    static neverRun(status: string) {
      if (status === this.SKIPPED || status === this.NEVER_BUILT || status === this.SKIPPED || status === this.DISABLED) {
        return true;
      }

      return false;
    }

    static isActive(status: string) {
      if (status === this.WAITING || status === this.BUILDING) {
        return true;
      }

      return false;
    }
}

export class PipelineAudit {
    id: number;
    user: User;
    versionned: Date;
    pipeline: Pipeline;
    action: string;
}

export class PipelineAuditDiff {
    type: string;
    before: any;
    after: any;
    title: string;
}

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
    usage: Usage;

    // true if someone has updated the pipeline ( used for warnings )
    externalChange: boolean;

    // UI Params
    forceRefresh: boolean;
    previewMode: boolean;

    // Return true if pattern is good
    public static checkName(name: string): boolean {
      if (!name) {
          return false;
      }

      if (!pipelineNamePattern.test(name)) {
          return false;
      }
      return true;
    }

    public static hasParameterWithoutValue(pipeline: Pipeline) {
        if (pipeline.parameters) {
            let emptyParams = pipeline.parameters.filter(p => {
                return !p.value || p.value === '';
            });
            return emptyParams.length > 0;
        }
        return false;
    }

    public static mergeAndKeepOld(ref: Array<Parameter>, current: Array<Parameter>): Array<Parameter> {
        if (!current) {
            return ref;
        }
        if (!ref) {
            return current;
        }

        let mapParam = current.reduce((m, o) => {
            m[o.name] = o;
            return m;
        }, {});
        ref.forEach( a => {
            if (!mapParam[a.name]) {
                current.push(a)
            }
        });

        return current;
    }
    /**
     * Merge parameters
     * @param ref
     * @param current
     */
    public static mergeParams(ref: Array<Parameter>, current: Array<Parameter>): Array<Parameter> {
        if (!ref) {
            return [];
        }

        if (!current || current.length === 0) {
            return ref;
        }

        return ref.map(p => {
            let idFound = current.findIndex((c) => c.name === p.name);

            return idFound === -1 ? p : current[idFound];
        });
    }

    constructor() {
        this.usage = new Usage();
    }
}

export class PipelineRunRequest {
    parameters: Array<Parameter>;
    env: Environment;
    parent_build_number: number; // instead of version
    parent_pipeline_id: number;
    parent_environment_id: number;
    parent_application_id: number;
    parent_version: number; // instead of build_number

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
    commits: Array<Commit>;
    warnings: Array<ActionWarning>;

    public static GetTriggerSource(pb: PipelineBuild): string {
        if (pb.trigger.scheduled_trigger) {
            return 'CDS scheduler';
        }
        if (pb.trigger.triggered_by && pb.trigger.triggered_by.username) {
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
    spawninfos: Array<SpawnInfo>;
    warnings: Array<ActionWarning>;
}

export class SpawnInfo {
    api_time: Date;
    remote_time: Date;
    user_message: string;
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
    val: string;
    start: LogDate;
    last_modified: LogDate;
    done: LogDate;
}

export interface ServiceLog {
    id: number;
    workflow_node_run_id: number;
    workflow_node_run_job_id: number;
    requirement_id: number;
    requirement_service_name: number;
    val: string;
    start: LogDate;
    last_modified: LogDate;
}

export class LogDate {
    seconds: number;
}

export class PipelineBuildTrigger {
    manual_trigger: boolean;
    scheduled_trigger: boolean;
    triggered_by: User;
    parent_pipeline_build: PipelineBuild;
    vcs_branch: string;
    vcs_hash: string;
    vcs_author: string;
    vcs_remote: string;
    vcs_remote_url: string;
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
    fullname: string;
    name: string;
    time: string;
    errors: Array<Failure>;
    failures: Array<Failure>;
    status: string;
    skipped: Array<Skipped>;
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

// Skipped contains data related to a skipped test.
export class Skipped {
    value: string;
}

// InnerResult is used by TestCase
export interface InnerResult {
    value: string;
}
