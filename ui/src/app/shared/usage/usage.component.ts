import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';

@Component({
    selector: 'app-usage',
    templateUrl: './usage.component.html',
    styleUrls: ['./usage.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class UsageComponent {

    @Input() project: Project;
    @Input() workflows: Array<Workflow>;
    @Input() applications: Array<Application>;
    @Input() pipelines: Array<Pipeline>;
    @Input() environments: Array<Environment>;

    constructor() { }
}
