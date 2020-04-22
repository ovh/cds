import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { ConfirmModalComponent } from 'app/shared/modal/confirm/confirm.component';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { PermissionEvent } from 'app/shared/permission/permission.event.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddGroupInProject, DeleteGroupInProject, UpdateGroupInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-permissions',
    templateUrl: './permission.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectPermissionsComponent {

    @Input() project: Project;

    @ViewChild('permWarning')
    permWarningModal: WarningModalComponent;
    @ViewChild('confirmPropagationModal')
    confirmPropagationModal: ConfirmModalComponent;

    loading = false;
    permFormLoading = false;
    currentPermEvent: PermissionEvent;

    constructor(
        public _translate: TranslateService,
        private _toast: ToastService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {

    }

    groupEvent(event: PermissionEvent, skip?: boolean): void {
        this.currentPermEvent = event;
        if (!skip && this.project.externalChange) {
            this.permWarningModal.show(event);
        } else {
            switch (event.type) {
                case 'add':
                    this.confirmPropagationModal.show();
                    break;
                case 'update':
                    this.store.dispatch(new UpdateGroupInProject({ projectKey: this.project.key, group: event.gp }))
                        .subscribe(() => this._toast.success('', this._translate.instant('permission_updated')));
                    break;
                case 'delete':
                    this.store.dispatch(new DeleteGroupInProject({ projectKey: this.project.key, group: event.gp }))
                        .subscribe(() => this._toast.success('', this._translate.instant('permission_deleted')));
                    break;
            }
        }
    }

    confirmPermPropagation(propagate: boolean) {
        this.permFormLoading = true;
        this.store.dispatch(new AddGroupInProject({
            projectKey: this.project.key,
            group: this.currentPermEvent.gp,
            onlyProject: !propagate
        })).pipe(finalize(() => {
            this.permFormLoading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => this._toast.success('', this._translate.instant('permission_added')));
    }
}
