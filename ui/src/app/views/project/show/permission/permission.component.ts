import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddGroupInProject, DeleteGroupInProject, UpdateGroupInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import { Warning } from '../../../../model/warning.model';
import { ConfirmModalComponent } from '../../../../shared/modal/confirm/confirm.component';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { PermissionEvent } from '../../../../shared/permission/permission.event.model';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-permissions',
    templateUrl: './permission.html'
})
export class ProjectPermissionsComponent {

    @Input() project: Project;
    @Input() warnings: Array<Warning>;

    @ViewChild('permWarning', {static: true})
    permWarningModal: WarningModalComponent;
    @ViewChild('confirmPropagationModal', {static: true})
    confirmPropagationModal: ConfirmModalComponent;

    permissionEnum = PermissionValue;
    loading = false;
    permFormLoading = false;
    currentPermEvent: PermissionEvent;

    constructor(
        public _translate: TranslateService,
        private _toast: ToastService,
        private store: Store
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
        })).pipe(finalize(() => this.permFormLoading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('permission_added')));
    }
}
