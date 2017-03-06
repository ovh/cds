import {Component, Input, ViewChild, Output, EventEmitter} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {Application} from '../../../../../model/application.model';
import {Scheduler} from '../../../../../model/scheduler.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Pipeline} from '../../../../../model/pipeline.model';
import {SchedulerEvent} from '../scheduler.event';

declare var _: any;

@Component({
    selector: 'app-application-scheduler-item',
    templateUrl: './scheduler.item.html',
    styleUrls: ['./scheduler.item.scss']
})
export class ApplicationSchedulerItemComponent {

    @Input() project: Project;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() scheduler: Scheduler;
    @Input() edit: boolean;

    editableScheduler: Scheduler;

    // Schedulers modals
    @ViewChild('editSchedulerModal')
    editSchedulerModal: SemanticModalComponent;

    @Output() event = new EventEmitter();

    constructor() { }

    editScheduler(): void {
        if (this.editSchedulerModal) {
            this.editableScheduler = _.cloneDeep(this.scheduler);
            this.editSchedulerModal.show();
        }
    }

    close(): void {
        if (this.editSchedulerModal) {
            this.editSchedulerModal.hide();
        }
    }

    updateScheduler(): void {
        this.scheduler.updating = true;
        this.event.emit(new SchedulerEvent('update', this.editableScheduler));
        this.close();
    }

    deleteScheduler(): void {
        this.scheduler.updating = true;
        this.event.emit(new SchedulerEvent('delete', this.editableScheduler));
        this.close();
    }
}
