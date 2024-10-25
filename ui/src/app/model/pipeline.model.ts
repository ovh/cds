import { AsCodeEvents } from './ascode.model';
import { Parameter } from './parameter.model';
import { Stage } from './stage.model';
import { Usage } from './usage.model';
import { Workflow } from './workflow.model';

export const pipelineNamePattern = new RegExp('^[a-zA-Z0-9._-]+$');

export class PipelineStatus {
    static BUILDING = 'Building';
    static FAIL = 'Fail';
    static SUCCESS = 'Success';
    static WAITING = 'Waiting';
    static DISABLED = 'Disabled';
    static SCHEDULING = 'Scheduling';
    static SKIPPED = 'Skipped';
    static NEVER_BUILT = 'Never Built';
    static STOPPED = 'Stopped';
    static PENDING = 'Pending';

    static priority = [
        PipelineStatus.NEVER_BUILT, PipelineStatus.SCHEDULING, PipelineStatus.PENDING, PipelineStatus.WAITING,
        PipelineStatus.BUILDING, PipelineStatus.STOPPED,
        PipelineStatus.FAIL, PipelineStatus.SUCCESS, PipelineStatus.DISABLED, PipelineStatus.SKIPPED
    ];

    static neverRun(status: string) {
        return status === this.SKIPPED || status === this.NEVER_BUILT || status === this.SKIPPED || status === this.DISABLED;
    }

    static isActive(status: string) {
        return status === this.WAITING || status === this.BUILDING || status === this.PENDING || status === this.SCHEDULING;
    }

    static isDone(status: string) {
        return status === this.SUCCESS || status === this.STOPPED || status === this.FAIL ||
            status === this.SKIPPED || status === this.DISABLED;
    }

    static sum(status: Array<string>): string {
        const sum = status.map(s => PipelineStatus.priority.indexOf(s)).reduce((sum, num) => {
            if (num > -1 && num < sum) { return num; }
            return sum;
        });
        if (sum === -1) {
            return null;
        }
        return PipelineStatus.priority[sum];
    }
}

export class PipelineAudit {
    id: number;
    username: string;
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
    description: string;
    icon: string;
    stages: Array<Stage>;
    parameters: Array<Parameter>;
    last_modified: number;
    projectKey: string;
    usage: Usage;
    audits: Array<PipelineAudit>;
    preview: Pipeline;
    asCode: string;
    from_repository: string;
    workflow_ascode_holder: Workflow;
    ascode_events: Array<AsCodeEvents>;

    // true if someone has updated the pipeline ( used for warnings )
    externalChange: boolean;

    // UI Params
    forceRefresh: boolean;
    previewMode: boolean;
    editModeChanged: boolean;

    constructor() {
        this.usage = new Usage();
    }

    // Return true if pattern is good
    public static checkName(name: string): boolean {
        if (!name) {
            return false;
        }

        return pipelineNamePattern.test(name);
    }

    public static hasParameterWithoutValue(pipeline: Pipeline) {
        if (pipeline.parameters) {
            let emptyParams = pipeline.parameters.filter(p => !p.value || p.value === '');
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
        ref.forEach(a => {
            if (!mapParam[a.name]) {
                current.push(a);
            }
        });

        return current;
    }
    /**
     * Merge parameters
     *
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

    static InitRef(editPipeline: Pipeline) {

        if (editPipeline && editPipeline.stages) {
            editPipeline.stages.forEach(s => {
                let nextRef;
                do {
                    nextRef = Math.random();
                } while (editPipeline.stages.findIndex(stg => stg.ref === nextRef) !== -1);
                s.ref = nextRef;
                if (s.jobs) {
                    s.jobs.forEach(j => {
                        let nextJobRef;
                        let loopAgain = false;
                        do {
                            loopAgain = false;
                            nextJobRef = Math.random();
                            stageLoop: for (let stageIndex = 0; stageIndex < editPipeline.stages.length; stageIndex++) {
                                let currentStage = editPipeline.stages[stageIndex];
                                if (!currentStage.jobs) {
                                    continue;
                                }
                                for (let jobIndex = 0; jobIndex < currentStage.jobs.length; jobIndex++) {
                                    let currentJob = currentStage[jobIndex];
                                    if (currentJob?.ref === nextRef) {
                                        loopAgain = true;
                                        break stageLoop;
                                    }
                                }
                            }
                        } while (loopAgain);
                        j.ref = nextJobRef;
                    });
                }
            });
        }
    }
}

export class SpawnInfo {
    api_time: Date;
    remote_time: Date;
    type: string;
    message: SpawnInfoMessage;
    user_message: string;
}

export class SpawnInfoMessage {
    args: Array<string>;
    id: string;
}

export class Tests {
    pipeline_build_id: number;
    total: number;
    ok: number;
    ko: number;
    skipped: number;
    test_suites: Array<TestSuite>;

    static getColor(t: string): string {
        switch (t) {
            case 'ok':
                return '#21BA45';
            case 'ko':
                return '#FF4F60';
            case 'skip':
                return '#808080';
        }
    }
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
    messages: string;
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
    message: string;
}

// InnerResult is used by TestCase
export interface InnerResult {
    value: string;
}
