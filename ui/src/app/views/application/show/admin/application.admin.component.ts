import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Project } from 'app/model/project.model';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { DeleteApplication, UpdateApplication } from 'app/store/applications.action';
import { FetchIntegrationsInProject } from 'app/store/project.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-application-admin',
    templateUrl: './application.admin.html',
    styleUrls: ['./application.admin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ApplicationAdminComponent implements OnInit {
    @Input() application: Application;
    @Input() project: Project;
    @Input() editMode: boolean;
    @ViewChild('updateWarning')
    private updateWarningModal: WarningModalComponent;

    newName: string;
    fileTooLarge = false;
    public loading = false;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _router: Router,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit() {
        // Fetch project integration
        this._store.dispatch(new FetchIntegrationsInProject({ projectKey: this.project.key }));

        this.newName = this.application.name;
        if (!this.project.permissions.writable) {
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
            this._store.dispatch(new UpdateApplication({
                projectKey: this.project.key,
                applicationName: this.application.name,
                changes: app
            })).pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
                .subscribe(() => {
                    if (this.editMode) {
                        this._toast.info('', this._translate.instant('application_ascode_updated'));
                    } else {
                        this._toast.success('', this._translate.instant('application_update_ok'));
                        if (nameUpdated) {
                            this._router.navigate(['/project', this.project.key, 'application', this.newName]);
                        }
                    }

                });
        }
    }

    deleteApplication(): void {
        this.loading = true;
        this._store.dispatch(new DeleteApplication({ projectKey: this.project.key, applicationName: this.application.name }))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('application_deleted'));
                this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
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
