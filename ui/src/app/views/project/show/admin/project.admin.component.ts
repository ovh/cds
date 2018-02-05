import {Input, ViewChild, OnInit, Component} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../../shared/toast/ToastService';
import {WarningModalComponent} from '../../../../shared/modal/warning/warning.component';
import {Router} from '@angular/router';
import {User} from '../../../../model/user.model';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';

@Component({
    selector: 'app-project-admin',
    templateUrl: './project.admin.html',
    styleUrls: ['./project.admin.scss']
})
export class ProjectAdminComponent implements OnInit {

    @Input() project: Project;
    @ViewChild('updateWarning')
        private warningUpdateModal: WarningModalComponent;

    loading = false;
    migrationValue = 0.0;
    user: User;

    constructor(private _projectStore: ProjectStore, private _toast: ToastService,
                public _translate: TranslateService, private _router: Router, private _authStore: AuthentificationStore) {};

    ngOnInit(): void {
        if (this.project.permission !== 7) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' }});
        }
        if (this.project.applications) {
            this.project.applications.forEach(app => {
                if (app.workflow_migration === 'STARTED') {
                    this.migrationValue += 0.5;
                } else if (app.workflow_migration === 'DONE' || app.workflow_migration === 'CLEANING') {
                    this.migrationValue++;
                }
            });
        }
        this.user = this._authStore.getUser();
    }

    onSubmitProjectUpdate(skip?: boolean) {
        if (!skip && this.project.externalChange) {
            this.warningUpdateModal.show();
        } else {
            this.loading = true;
            this._projectStore.updateProject(this.project).subscribe(() => {
                this.loading = false;
                this._toast.success('', this._translate.instant('project_update_msg_ok') );
            }, () => {
                this.loading = false;
            });
        }
    };

    deleteProject(): void {
        this._projectStore.deleteProject(this.project.key).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('project_deleted'));
            this._router.navigate(['/home']);
        }, () => {
            this.loading = false;
        });
    }

}
