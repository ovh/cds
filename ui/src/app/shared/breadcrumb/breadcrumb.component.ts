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
        if (!this.application && this.pipeline) {
            queryParams['tab'] = 'pipelines';
        } else {
            queryParams['tab'] = 'applications';
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
