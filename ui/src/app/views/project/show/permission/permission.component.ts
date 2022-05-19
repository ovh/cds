import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { GroupPermission } from 'app/model/group.model';
import { Project } from 'app/model/project.model';
import { ConfirmModalComponent } from 'app/shared/modal/confirm/confirm.component';
import { PermissionEvent } from 'app/shared/permission/permission.event.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddGroupInProject, DeleteGroupInProject, UpdateGroupInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';
import cloneDeep from 'lodash-es/cloneDeep';

@Component({
    selector: 'app-project-permissions',
    templateUrl: './permission.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectPermissionsComponent implements OnChanges {
    @ViewChild('confirmPropagationModal') confirmPropagationModal: ConfirmModalComponent;

    groups: Array<GroupPermission> = [];
    _project: Project;
    @Input() set project(data: Project) {
        this._project = data;
        if (data) {
            this.groups = cloneDeep(data.groups);
        }
    };

    get project() {
        return this._project;
    }

    loading = false;
    permFormLoading = false;
    currentPermEvent: PermissionEvent;
    groupsOutsideOrganization: Array<GroupPermission>;

    constructor(
        public _translate: TranslateService,
        private _toast: ToastService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {
    }

    ngOnChanges(): void {
        if (this.project && !!this.project.organization) {
            this.groupsOutsideOrganization = this.project.groups.filter(gp =>
                gp.group.organization && gp.group.organization !== this.project.organization);
            this._cd.markForCheck();
        }
    }

    groupEvent(event: PermissionEvent): void {
        this.currentPermEvent = event;

        switch (event.type) {
            case 'add':
                this.confirmPropagationModal.show();
                break;
            case 'update':
                this.store.dispatch(new UpdateGroupInProject({projectKey: this.project.key, group: event.gp}))
                    .pipe(finalize(() => {
                        this.groups = cloneDeep(this.project.groups);
                        this._cd.markForCheck();
                    }))
                    .subscribe(() => this._toast.success('', this._translate.instant('permission_updated')));
                break;
            case 'delete':
                this.store.dispatch(new DeleteGroupInProject({projectKey: this.project.key, group: event.gp}))
                    .pipe(finalize(() => {
                        this.groups = cloneDeep(this.project.groups);
                        this._cd.markForCheck();
                    }))
                    .subscribe(() => this._toast.success('', this._translate.instant('permission_deleted')));
                break;

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
