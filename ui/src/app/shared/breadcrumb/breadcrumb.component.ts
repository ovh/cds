import {Component, Input} from '@angular/core';
import {Project} from '../../model/project.model';
import {Application} from '../../model/application.model';
import {Pipeline} from '../../model/pipeline.model';
import {Action} from '../../model/action.model';
import {Router} from '@angular/router';

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

    constructor(private _router: Router) { }

    navigateToProject(): void {
        let queryParams = { queryParams: {}};
        if (!this.application && this.pipeline) {
            queryParams.queryParams['tab'] = 'pipelines';
        } else {
            queryParams.queryParams['tab'] = 'applications';
        }
        this._router.navigate(['project', this.project.key], queryParams);
    }

    navigateToApplication(): void {
        let queryParams = { queryParams: {}};
        if (this.branch) {
            queryParams.queryParams['branch'] = this.branch;
        }
        if (this.version) {
            queryParams.queryParams['version'] = this.version;
        }
        this._router.navigate(['project', this.project.key, 'application', this.application.name], queryParams);
    }

    navigateToPipeline(): void {
        let queryParams = { queryParams: {}};
        if (this.application) {
            queryParams.queryParams['application'] = this.application.name;
        }
        if (this.version) {
            queryParams.queryParams['version'] = this.version;
        }
        if (this.buildNumber) {
            queryParams.queryParams['buildNumber'] = this.buildNumber;
        }
        if (this.envName) {
            queryParams.queryParams['envName'] = this.envName;
        }
        if (this.branch) {
            queryParams.queryParams['branch'] = this.branch;
        }
        this._router.navigate(['project', this.project.key, 'pipeline', this.pipeline.name], queryParams);
    }

    navigateToBuild(): void {
        let queryParams = { queryParams: {}};
        queryParams.queryParams['envName'] = this.envName;
        this._router.navigate([
            '/project',  this.project.key,
            'application', this.application.name,
            'pipeline', this.pipeline.name,
            'build', this.buildNumber
        ], queryParams);
    }
}
