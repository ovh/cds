import {GroupPermission} from './group.model';
import {PipelineBuild, Pipeline} from './pipeline.model';
import {Notification} from './notification.model';
import {Variable} from './variable.model';
import {Parameter} from './parameter.model';
import {RepositoriesManager} from './repositories.model';
import {RepositoryPoller} from './polling.model';
import {Hook} from './hook.model';
import {WorkflowItem} from './application.workflow.model';

export class Application {
    id: number;
    name: string;
    groups: Array<GroupPermission>;
    variables: Array<Variable>;
    pipelines: Array<ApplicationPipeline>;
    pipelines_build: Array<PipelineBuild>;
    permission: number;
    notifications: Array<Notification>;
    last_modified: number;
    repositories_manager: RepositoriesManager;
    repository_fullname: string;
    pollers: Array<RepositoryPoller>;
    hooks: Array<Hook>;
    workflows: Array<WorkflowItem>;

    project_key: string; // project unique key

    // true if someone has updated the application ( used for warnings )
    externalChange: boolean;
}

export class ApplicationPipeline {
    pipeline: Pipeline;
    parameters: Array<Parameter>;
    last_modified: number;
}

export interface ApplicationFilter {
    branch: string;
    version: number;
};
