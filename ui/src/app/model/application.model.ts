import { Key } from './keys.model';
import { Metric } from './metric.model';
import { Notification } from './notification.model';
import { Usage } from './usage.model';
import { Variable } from './variable.model';
import { VCSStrategy } from './vcs.model';
import { WorkflowRun } from './workflow.run.model';

export const applicationNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');

export class Application {
    id: number;
    name: string;
    description: string;
    icon: string;
    variables: Array<Variable>;
    notifications: Array<Notification>;
    last_modified: string;
    vcs_server: string;
    repository_fullname: string;
    usage: Usage;
    keys: Array<Key>;
    vcs_strategy: VCSStrategy;
    deployment_strategies: Map<string, any>;
    vulnerabilities: Array<Vulnerability>;
    project_key: string; // project unique key
    from_repository: string;

    // true if someone has updated the application ( used for warnings )
    externalChange: boolean;

    // workflow depth for horizontal tree view
    horizontalDepth: number;

    // Return true if pattern is good
    public static checkName(name: string): boolean {
        if (!name) {
            return false;
        }

        if (!applicationNamePattern.test(name)) {
            return false;
        }
        return true;
    }
}

export class Vulnerability {
    id: number;
    application_id: number;
    title: string;
    description: string;
    cve: string;
    link: string;
    component: string;
    version: string;
    origin: string;
    severity: string;
    fix_in: string;
    ignored: boolean;

    // ui param
    loading: boolean;
}

export class Severity {
    static UNKNOWN = 'unknown';
    static NEGLIGIBLE = 'negligible';
    static LOW = 'low';
    static MEDIUM = 'medium';
    static HIGH = 'high';
    static CRITICAL = 'critical';
    static DEFCON1 = 'defcon1';

    static Severities = [
        Severity.UNKNOWN,
        Severity.NEGLIGIBLE,
        Severity.LOW,
        Severity.MEDIUM,
        Severity.HIGH,
        Severity.CRITICAL,
        Severity.DEFCON1
    ];

    static getColors(s: string) {
        switch (s) {
            case Severity.DEFCON1:
                return '#000000';
            case Severity.CRITICAL:
                return '#8B0000';
            case Severity.HIGH:
                return '#FF4F60';
            case Severity.MEDIUM:
                return '#FFA500';
            case Severity.LOW:
                return '#21BA45';
            case Severity.NEGLIGIBLE:
                return '#808080';
            case Severity.UNKNOWN:
                return '#D3D3D3';
        }
    }
}

export class Overview {
    graphs: Array<OverviewGraph>;
    git_url: string;
    history: { [key: string]: Array<WorkflowRun>; };
}

export class OverviewGraph {
    type: string;
    datas: Array<Metric>;
}
