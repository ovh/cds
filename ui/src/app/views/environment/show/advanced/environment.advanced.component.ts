import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { Project } from 'app/model/project.model';
import { User } from 'app/model/user.model';
import { AuthentificationStore } from 'app/service/auth/authentification.store';
import { ToastService } from 'app/shared/toast/ToastService';
import {
    CloneEnvironmentInProject,
    DeleteEnvironmentInProject,
    UpdateEnvironmentInProject
} from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-environment-advanced',
    templateUrl: './environment.advanced.html',
    styleUrls: ['./environment.advanced.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class EnvironmentAdvancedComponent implements OnInit {

    @Input() environment: Environment;
    @Input() project: Project;

    user: User;

    oldName: string;
    fileTooLarge = false;
    cloneName: string;
    public loading = false;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _router: Router,
        private _authStore: AuthentificationStore,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) {

    }

    ngOnInit() {
        this.user = this._authStore.getUser();
        this.oldName = this.environment.name;
        if (this.environment.permission !== 7) {
            this._router.navigate(['/project', this.project.key, 'environment', this.environment.name]);
        }
    }

    onSubmitEnvironmentUpdate(): void {
        this.loading = true;
        this.store.dispatch(new UpdateEnvironmentInProject({
            projectKey: this.project.key,
            environmentName: this.oldName,
            changes: this.environment
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('environment_renamed'));
                this._router.navigate(['/project', this.project.key, 'environment', this.environment.name]);
            });
    }

    cloneEnvironment(cloneModal?: any): void {
        this.loading = true;
        this.store.dispatch(new CloneEnvironmentInProject({
            projectKey: this.project.key,
            cloneName: this.cloneName,
            environment: this.environment
        })).pipe(finalize(() => {
            this.loading = false;
            this.cloneName = '';
            cloneModal.hide();
            this._cd.markForCheck();
        })).subscribe(() => {
            this._toast.success('', this._translate.instant('environment_cloned'));
            this._router.navigate(['/project', this.project.key, 'environment', this.cloneName]);
        });
    }

    deleteEnvironment(): void {
        this.loading = true;
        this.store.dispatch(new DeleteEnvironmentInProject({
            projectKey: this.project.key, environment: this.environment
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('environment_deleted'));
                this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'environments' } });
            });
    }
}
