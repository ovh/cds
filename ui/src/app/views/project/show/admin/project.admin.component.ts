import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { APIConfig } from 'app/model/config.service';
import { Project } from 'app/model/project.model';
import { FeatureNames, FeatureService } from 'app/service/feature/feature.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { ConfigState } from 'app/store/config.state';
import { DeleteProject, UpdateProject } from 'app/store/project.action';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-admin',
    templateUrl: './project.admin.html',
    styleUrls: ['./project.admin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectAdminComponent implements OnInit {

    @Input() project: Project;

    loading = false;
    fileTooLarge = false;
    configSubscription: Subscription;
    apiConfig: APIConfig;
    v2Enabled: boolean = false;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _router: Router,
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _featureService: FeatureService
    ) { }

    ngOnInit(): void {
        this._featureService.isEnabled(FeatureNames.AllAsCode, { project_key: this.project.key }).subscribe(f => {
            this.v2Enabled = f.enabled;
            this._cd.markForCheck();
        });
        if (!this.project.permissions.writable) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
        }

        this.configSubscription = this._store.select(ConfigState.api).subscribe(c => {
            this.apiConfig = c;
            this._cd.markForCheck();
        });
    }

    onSubmitProjectUpdate() {
        this.loading = true;
        this._store.dispatch(new UpdateProject(this.project))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', this._translate.instant('project_update_msg_ok')));
    }

    deleteProject(): void {
        this.loading = true;
        this._store.dispatch(new DeleteProject({ projectKey: this.project.key }))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('project_deleted'));
                this._router.navigate(['/']);
            });
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000;
        if (this.fileTooLarge) {
            return;
        }
        this.project.icon = event.content;
    }
}
