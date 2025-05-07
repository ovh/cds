import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { ProjectIntegration } from 'app/model/integration.model';
import { Project } from 'app/model/project.model';
import { ProjectService } from 'app/service/project/project.service';
import { ErrorUtils } from 'app/shared/error.utils';
import { ToastService } from 'app/shared/toast/ToastService';
import { DeleteApplication, UpdateApplication } from 'app/store/applications.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { NzMessageService } from 'ng-zorro-antd/message';
import { lastValueFrom } from 'rxjs';
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

    newName: string;
    fileTooLarge = false;
    loading = false;
    deploymentIntegrations: Array<ProjectIntegration> = [];

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _router: Router,
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _messageService: NzMessageService,
        private _projectService: ProjectService
    ) { }

    ngOnInit() {
        this.newName = this.application.name;
        if (!this.project.permissions.writable) {
            this._router.navigate(['/project', this.project.key, 'application', this.application.name],
                { queryParams: { tab: 'workflow' } });
        }
        this.load();
    }

    async load() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            const projectIntegrations = await lastValueFrom(this._projectService.getIntegrations(this.project.key));
            this.deploymentIntegrations = projectIntegrations.filter(p => p.model.deployment);
        } catch (e) {
            this._messageService.error(`Unable to load project integrations: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }

    onSubmitApplicationUpdate(): void {
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

    deleteApplication(): void {
        this.loading = true;
        this._store.dispatch(new DeleteApplication({
            projectKey: this.project.key,
            applicationName: this.application.name
        }))
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
