import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { finalize, first } from 'rxjs/operators';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import { Warning } from '../../../../model/warning.model';
import { ProjectStore } from '../../../../service/project/project.store';
import { ConfirmModalComponent } from '../../../../shared/modal/confirm/confirm.component';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { PermissionEvent } from '../../../../shared/permission/permission.event.model';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-permissions',
    templateUrl: './permission.html'
})
export class ProjectPermissionsComponent implements OnInit {

    @Input() project: Project;
    @Input() warnings: Array<Warning>;

    @ViewChild('permWarning')
    permWarningModal: WarningModalComponent;
    @ViewChild('confirmPropagationModal')
    confirmPropagationModal: ConfirmModalComponent;

    permissionEnum = PermissionValue;
    loading = true;
    permFormLoading = false;
    currentPermEvent: PermissionEvent;

    constructor(private _projectStore: ProjectStore,
        public _translate: TranslateService,
        private _toast: ToastService) {

    }

    ngOnInit() {
        if (this.project.environments) {
            this.loading = false;
            return
        }
        this._projectStore.getProjectEnvironmentsResolver(this.project.key)
            .pipe(first(), finalize(() => this.loading = false))
            .subscribe((proj) => {
                this.project = proj;
            });
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
                    this._projectStore.updateProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_updated'));
                    });
                    break;
                case 'delete':
                    this._projectStore.removeProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_deleted'));
                    });
                    break;
            }
        }
    }

    confirmPermPropagation(propagate: boolean) {
        this.permFormLoading = true;
        this._projectStore.addProjectPermission(this.project.key, this.currentPermEvent.gp, !propagate)
            .pipe(finalize(() => this.permFormLoading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('permission_added')));
    }
}
