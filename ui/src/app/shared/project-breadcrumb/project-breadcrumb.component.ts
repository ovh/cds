import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Action } from 'app/model/action.model';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';

@Component({
    selector: 'app-project-breadcrumb',
    templateUrl: './project-breadcrumb.html',
    styleUrls: ['./project-breadcrumb.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectBreadcrumbComponent {

    @Input() project: Project;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() environment: Environment;
    @Input() action: Action;
    @Input() version = 0;
    @Input() buildNumber = 0;
    @Input() envName: string;
    @Input() branch: string;
    @Input() remote: string;
    @Input() appVersion: number;
    @Input() workflow: string;
    @Input() workflowRun: string;
    @Input() workflowRunNode: string;
    @Input() wPipeline: string;

    constructor() {
    }

    getProjectQueryParams(): {} {
        let queryParams = {};
        if (this.pipeline) {
            queryParams['tab'] = 'pipelines';
        } else if (this.application) {
            queryParams['tab'] = 'applications';
        } else if (this.workflow) {
            queryParams['tab'] = 'workflows';
        } else if (this.environment) {
            queryParams['tab'] = 'environments';
        }

        return queryParams;
    }

    getApplicationQueryParams(): {} {
        let queryParams = {
            remote: this.remote || '',
            branch: this.branch || ''
        };

        return queryParams;
    }

    getPipelineQueryParams(): {} {
        let queryParams = {
            version: this.version || '',
            remote: this.remote || '',
            buildNumber: this.buildNumber || '',
            envName: this.envName || '',
            branch: this.branch || ''
        };

        if (this.application) {
            queryParams['application'] = this.application.name;
        }

        return queryParams;
    }

    getBuildQueryParams(): {} {
        let queryParams = {
            envName: this.envName || '',
            branch: this.branch || '',
            remote: this.remote || ''
        };

        return queryParams;
    }
}
