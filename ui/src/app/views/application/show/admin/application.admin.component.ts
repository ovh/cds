import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { DeleteApplication, UpdateApplication } from 'app/store/applications.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize } from 'rxjs/operators';
import { Application } from '../../../../model/application.model';
import { Project } from '../../../../model/project.model';
import { User } from '../../../../model/user.model';
import { AuthentificationStore } from '../../../../service/authentication/authentification.store';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-application-admin',
    templateUrl: './application.admin.html',
    styleUrls: ['./application.admin.scss']
})
export class ApplicationAdminComponent implements OnInit {

    @Input() application: Application;
    @Input() project: Project;
    @ViewChild('updateWarning', {static: false})
    private updateWarningModal: WarningModalComponent;

    user: User;

    newName: string;
    fileTooLarge = false;
    public loading = false;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _router: Router,
        private _authStore: AuthentificationStore,
        private store: Store
    ) {

    }

    ngOnInit() {
        this.user = this._authStore.getUser();
        this.newName = this.application.name;
        if (this.application.permission !== 7) {
            this._router.navigate(['/project', this.project.key, 'application', this.application.name],
                { queryParams: { tab: 'workflow' } });
        }
    }

    onSubmitApplicationUpdate(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.updateWarningModal.show();
        } else {
            this.loading = true;
            let nameUpdated = this.application.name !== this.newName;
            let app = cloneDeep(this.application);
            app.name = this.newName;
            this.store.dispatch(new UpdateApplication({
                projectKey: this.project.key,
                applicationName: this.application.name,
                changes: app
            })).pipe(finalize(() => this.loading = false))
                .subscribe(() => {
                    this._toast.success('', this._translate.instant('application_update_ok'));
                    if (nameUpdated) {
                        this._router.navigate(['/project', this.project.key, 'application', this.newName]);
                    }
                });
        }
    }

    deleteApplication(): void {
        this.loading = true;
        this.store.dispatch(new DeleteApplication({ projectKey: this.project.key, applicationName: this.application.name }))
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('application_deleted'));
                this._router.navigate(['/project', this.project.key], { queryParams: { 'tab': 'applications' } });
            });
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000;
        if (this.fileTooLarge) {
            return;
        }
        this.application.icon = event.content;
    }
}
