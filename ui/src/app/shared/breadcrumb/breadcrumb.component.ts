import {Component, Input} from '@angular/core';
import {Project} from '../../model/project.model';
import {Application} from '../../model/application.model';
import {Pipeline} from '../../model/pipeline.model';
import {Action} from '../../model/action.model';

@Component({
    selector: 'app-breadcrumb',
    templateUrl: './breadcrumb.html'
})
export class BreadcrumbComponent {

    @Input() project: Project;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() action: Action;
    @Input() version = 0;
    @Input() buildNumber = 0;
    @Input() envName: string;
    @Input() branch: string;
    @Input() appVersion: number;
    @Input() workflow: string;
    @Input() workflowRun: string;
    @Input() workflowRunNode: string;
    @Input() wPipeline: string;

    constructor() {
    }

    getProjectQueryParams(): {} {
        let queryParams = {};
        if (!this.application && this.pipeline) {
            queryParams['tab'] = 'pipelines';
        } else {
            queryParams['tab'] = 'applications';
        }

        return queryParams;
    }

    getApplicationQueryParams(): {} {
        let queryParams = {};
        if (this.branch) {
            queryParams['branch'] = this.branch;
        }
        return queryParams;
    }

    getPipelineQueryParams(): {} {
        let queryParams = {};
        if (this.application) {
            queryParams['application'] = this.application.name;
        }
        if (this.version) {
            queryParams['version'] = this.version;
        }
        if (this.buildNumber) {
            queryParams['buildNumber'] = this.buildNumber;
        }
        if (this.envName) {
            queryParams['envName'] = this.envName;
        }
        if (this.branch) {
            queryParams['branch'] = this.branch;
        }
        return queryParams;
    }

    getBuildQueryParams(): {} {
        let queryParams = {};
        queryParams['envName'] = this.envName;
        queryParams['branch'] = this.branch;
        return queryParams;
    }
}
