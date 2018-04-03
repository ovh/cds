import {Component, OnInit, ViewChild, Input} from '@angular/core';
import {Application} from '../../../../model/application.model';
import {ApplicationStore} from '../../../../service/application/application.store';
import {TranslateService} from '@ngx-translate/core';
import {Router} from '@angular/router';
import {ToastService} from '../../../../shared/toast/ToastService';
import {Project} from '../../../../model/project.model';
import {WarningModalComponent} from '../../../../shared/modal/warning/warning.component';
import {ApplicationMigrateService} from '../../../../service/application/application.migration.service';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {User} from '../../../../model/user.model';
import {finalize, first} from 'rxjs/operators';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-application-admin',
    templateUrl: './application.admin.html',
    styleUrls: ['./application.admin.scss']
})
export class ApplicationAdminComponent implements OnInit {

    @Input() application: Application;
    @Input() project: Project;
    @ViewChild('updateWarning')
        private updateWarningModal: WarningModalComponent;

    @ViewChild('doneMigrationTmpl')
    doneMigrationTmpl: ModalTemplate<boolean, boolean, void>;
    migrationModal: ActiveModal<boolean, boolean, void>;
    migrationText: string;

    user: User;

    newName: string;
    public loading = false;

    constructor(private _applicationStore: ApplicationStore, private _toast: ToastService, private _modalService: SuiModalService,
                public _translate: TranslateService, private _router: Router, private _appMigrateSerivce: ApplicationMigrateService,
                private _authStore: AuthentificationStore) {
    }

    ngOnInit() {
        this.user = this._authStore.getUser();
        this.newName = this.application.name;
        if (this.application.permission !== 7) {
            this._router.navigate(['/project', this.project.key, 'application', this.application.name],
                { queryParams: {tab: 'workflow'}});
        }
        this.migrationText = this._translate.instant('application_workflow_migration_modal_content');
    }

    generateWorkflow(force: boolean): void {
        this._appMigrateSerivce.migrateApplicationToWorkflow(this.project.key, this.application.name, force)
            .pipe(
              first(),
              finalize(() => this.loading = true)
            )
            .subscribe(() => {
            this._toast.success('', this._translate.instant('application_workflow_migration_success'));
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows'} });
        });
    }

    onSubmitApplicationUpdate(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.updateWarningModal.show();
        } else {
            this.loading = true;
            let app = cloneDeep(this.application);
            app.name = this.newName;
            this._applicationStore.updateApplication(this.project.key, this.application.name, app)
                .pipe(first()).subscribe( () => {
                this.loading = false;
                this._toast.success('', this._translate.instant('application_update_ok'));
                this._router.navigate(['/project', this.project.key, 'application', this.newName]);
            }, () => {
                this.loading = false;
            });
        }
    }

    openDoneMigrationPopup(): void {
        let tmpl = new TemplateModalConfig<boolean, boolean, void>(this.doneMigrationTmpl);
        this.migrationModal = this._modalService.open(tmpl);
    }

    migrationClean(): void {
        this.loading = true;
        this._appMigrateSerivce.cleanWorkflow(this.project.key, this.application.name).pipe(finalize(() => {
            this.loading = false;
        })).subscribe(() => {
           this._toast.success('', this._translate.instant('application_workflow_migration_ok'));
           this.migrationModal.approve(true);
        });
    }

    deleteApplication(): void {
        this.loading = true;
        this._applicationStore.deleteApplication(this.project.key, this.application.name).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('application_deleted'));
            this._router.navigate(['/project', this.project.key]);
        }, () => {
            this.loading = false;
        });
    }
}
