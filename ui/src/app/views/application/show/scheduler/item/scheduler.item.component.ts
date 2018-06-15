import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Application} from '../../../../../model/application.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {Project} from '../../../../../model/project.model';
import {Scheduler} from '../../../../../model/scheduler.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {ToastService} from '../../../../../shared/toast/ToastService';

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

    show = false;

    constructor(private _appStore: ApplicationStore, private _toast: ToastService, private _translate: TranslateService) {

    }

    editScheduler(): void {
        if (this.editSchedulerModal) {
            this.editableScheduler = cloneDeep(this.scheduler);
            this.show = true;
            setTimeout(() => {
                this.editSchedulerModal.show();
            }, 100);
        }
    }

    close(): void {
        if (this.editSchedulerModal) {
            this.show = false;
            this.editSchedulerModal.hide();
        }
    }

    updateScheduler(): void {
        this.scheduler.updating = true;
        this._appStore.updateScheduler(this.project.key, this.application.name, this.pipeline.name, this.editableScheduler)
            .subscribe(() => {
                this._toast.success('', this._translate.instant('scheduler_updated'));
                this.close();
            }, () => {
                this.scheduler.updating = false;
            });
    }

    deleteScheduler(): void {
        this.scheduler.updating = true;
        this._appStore.deleteScheduler(this.project.key, this.application.name, this.pipeline.name, this.editableScheduler)
            .subscribe(() => {
                this._toast.success('', this._translate.instant('scheduler_deleted'));
                this.close();
            }, () => {
                this.scheduler.updating = false;
            });
    }
}
