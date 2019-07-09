import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Application } from 'app/model/application.model';
import { Project } from 'app/model/project.model';

@Component({
    selector: 'app-usage-applications',
    templateUrl: './usage.applications.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class UsageApplicationsComponent {
    @Input() project: Project;
    @Input() applications: Array<Application>;

    constructor() { }
}
