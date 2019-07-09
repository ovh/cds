import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Environment } from 'app/model/environment.model';
import { Project } from 'app/model/project.model';

@Component({
    selector: 'app-usage-environments',
    templateUrl: './usage.environments.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class UsageEnvironmentsComponent {

    @Input() project: Project;
    @Input() environments: Array<Environment>;

    constructor() { }
}
