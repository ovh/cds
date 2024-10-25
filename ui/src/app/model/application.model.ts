import { Key } from './keys.model';
import { Metric } from './metric.model';
import { Usage } from './usage.model';
import { Variable } from './variable.model';
import { VCSStrategy } from './vcs.model';
import { Notification, Workflow } from './workflow.model';
import { WorkflowRunSummary } from './workflow.run.model';

export const applicationNamePattern = new RegExp('^[a-zA-Z0-9._-]+$');

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
    deployment_strategies: {};
    project_key: string; // project unique key
    from_repository: string;
    overview: any;

    // true if someone has updated the application ( used for warnings )
    externalChange: boolean;
    editModeChanged: boolean;
    workflow_ascode_holder: Workflow;

    // Return true if pattern is good
    public static checkName(name: string): boolean {
        if (!name) {
            return false;
        }

        return applicationNamePattern.test(name);

    }
}

export class Overview {
    graphs: Array<OverviewGraph>;
    git_url: string;
    history: { [key: string]: Array<WorkflowRunSummary>; };
}

export class OverviewGraph {
    type: string;
    datas: Array<Metric>;
}
