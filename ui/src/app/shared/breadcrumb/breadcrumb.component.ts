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

    constructor(private _router: Router) { }

    navigateToProject(): void {
        this._router.navigate(['project', this.project.key]);
    }

    navigateToApplication(): void {
        this._router.navigate(['project', this.project.key, 'application', this.application.name]);
    }

    navigateToPipeline(): void {
        if (this.application) {
            this._router.navigate(['project', this.project.key, 'pipeline', this.pipeline.name]);
        } else {
            this._router.navigate(['project', this.project.key, 'pipeline', this.pipeline.name]);
        }

    }
}
