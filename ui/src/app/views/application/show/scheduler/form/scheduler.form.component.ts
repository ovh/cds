import {Component, Input} from '@angular/core';
import {Scheduler} from '../../../../../model/scheduler.model';
import {Project} from '../../../../../model/project.model';
import {Application} from '../../../../../model/application.model';
import {Pipeline} from '../../../../../model/pipeline.model';

declare var _: any;

@Component({
    selector: 'app-application-scheduler-form',
    templateUrl: './scheduler.form.html',
    styleUrls: ['./scheduler.form.scss']
})
export class ApplicationSchedulerFormComponent {

    @Input() application: Application;
    @Input() project: Project;
    @Input() edit: boolean;
    @Input() scheduler: Scheduler;
    @Input() pipeline: Pipeline;

    constructor() {
    }
}
