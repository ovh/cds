import { Component, Input, OnInit, ViewChild } from '@angular/core';
import {WarningModalComponent} from '../../../../shared/modal/warning/warning.component';
import {Project} from '../../../../model/project.model';
import {PermissionValue} from '../../../../model/permission.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {ToastService} from '../../../../shared/toast/ToastService';
import {EnvironmentPermissionEvent, PermissionEvent} from '../../../../shared/permission/permission.event.model';
import {Environment} from '../../../../model/environment.model';
import {TranslateService} from '@ngx-translate/core';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-project-permissions',
    templateUrl: './permission.html'
})
export class ProjectPermissionsComponent implements OnInit {

    @Input() project: Project;

    @ViewChild('permWarning')
    permWarningModal: WarningModalComponent;
    @ViewChild('permEnvWarning')
    permEnvWarningModal: WarningModalComponent;
    @ViewChild('permEnvGroupWarning')
    permEnvGroupWarningModal: WarningModalComponent;

    permissionEnum = PermissionValue;
    loading = true;
    permFormLoading = false;
    permEnvFormLoading = false

    constructor(
      private _projectStore: ProjectStore,
      public _translate: TranslateService,
      private _toast: ToastService) {

    }

    ngOnInit() {
      this._projectStore.getProjectEnvironmentsResolver(this.project.key)
        .pipe(finalize(() => this.loading = false))
        .subscribe((proj) => {
          this.project = proj;
        });
    }

    addEnvPermEvent(event: EnvironmentPermissionEvent, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.permEnvWarningModal.show(event);
        } else {
            this.permEnvFormLoading = true;
            this._projectStore.addEnvironmentPermission(this.project.key, event.env.name, event.gp).subscribe((proj) => {
                this._toast.success('', this._translate.instant('permission_added'));
                this.project = proj;
                this.permEnvFormLoading = false;
            }, () => {
                this.permEnvFormLoading = false;
            });
        }
    }

    envGroupEvent(event: PermissionEvent, env: Environment, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            event.env = env;
            this.permEnvGroupWarningModal.show(event);
        } else {
            if (!env) {
                env = event.env;
            }
            switch (event.type) {
                case 'update':
                    this._projectStore.updateEnvironmentPermission(this.project.key, env.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_updated'));
                    });
                    break;
                case 'delete':
                    this._projectStore.removeEnvironmentPermission(this.project.key, env.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_deleted'));
                    });
                    break;
            }
        }
    }

    groupEvent(event: PermissionEvent, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.permWarningModal.show(event);
        } else {
            switch (event.type) {
                case 'add':
                    this.permFormLoading = true;
                    this._projectStore.addProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_added'));
                        this.permFormLoading = false;
                    }, () => {
                        this.permFormLoading = false;
                    });
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
}
