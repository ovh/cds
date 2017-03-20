import {Component, Input, ViewChild, Output, EventEmitter} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {Application} from '../../../../../model/application.model';
import {Scheduler} from '../../../../../model/scheduler.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Pipeline} from '../../../../../model/pipeline.model';
import {RepositoryPoller} from '../../../../../model/polling.model';

declare var _: any;

@Component({
    selector: 'app-application-poller-item',
    templateUrl: './poller.item.html',
    styleUrls: ['./poller.item.scss']
})
export class ApplicationPollerItemComponent {

    @Input() project: Project;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() poller: RepositoryPoller;

    editableScheduler: Scheduler;

    // Schedulers modals
    @ViewChild('editSchedulerModal')
    editSchedulerModal: SemanticModalComponent;

    @Output() event = new EventEmitter();

    constructor() { }

    /*
    editScheduler(): void {
        if (this.editSchedulerModal) {
            this.editableScheduler = _.cloneDeep(this.scheduler);
            setTimeout(() => {
                this.editSchedulerModal.show();
            }, 100);
        }
    }

    close(): void {
        if (this.editSchedulerModal) {
            this.editSchedulerModal.hide();
        }
    }

    updatePoller(): void {
        this.scheduler.updating = true;
        this.event.emit(new SchedulerEvent('update', this.editableScheduler));
        this.close();
    }

    deletePoller(): void {
        this.scheduler.updating = true;
        this.event.emit(new SchedulerEvent('delete', this.editableScheduler));
        this.close();
    }
    */
}
