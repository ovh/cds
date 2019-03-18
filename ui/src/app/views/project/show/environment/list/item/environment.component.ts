import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import * as projectActions from 'app/store/project.action';
import { cloneDeep } from 'lodash';
import { finalize } from 'rxjs/operators';
import { Application } from '../../../../../../model/application.model';
import { Environment } from '../../../../../../model/environment.model';
import { Pipeline } from '../../../../../../model/pipeline.model';
import { Project } from '../../../../../../model/project.model';
import { User } from '../../../../../../model/user.model';
import { Workflow } from '../../../../../../model/workflow.model';
import { AuthentificationStore } from '../../../../../../service/auth/authentification.store';
import { ToastService } from '../../../../../../shared/toast/ToastService';
import { VariableEvent } from '../../../../../../shared/variable/variable.event.model';

@Component({
    selector: 'app-environment',
    templateUrl: './environment.html',
    styleUrls: ['./environment.scss']
})
export class ProjectEnvironmentComponent implements OnInit {

    editableEnvironment: Environment;
    attachedWorkflows: Array<Workflow> = [];
    attachedPipelines: Array<Pipeline> = [];
    attachedApplications: Array<Application> = [];
    oldEnvName: string;
    cloneName: string;
    currentUser: User;

    hasChanged = false;
    loading = false;
    loadingUsage = true;
    cloneLoading = false;
    addVarLoading = false;

    @Input('environment')
    set environment(data: Environment) {
        if (!data) {
            return;
        }
        let oldName = this.oldEnvName;
        this.oldEnvName = data.name;
        this.editableEnvironment = cloneDeep(data);
        if (oldName !== data.name) {
            this.fetchUsage();
        }
        if (data.usage) {
            this.attachedWorkflows = data.usage.workflows || [];
            this.attachedApplications = data.usage.applications || [];
            this.attachedPipelines = data.usage.pipelines || [];
        }
    }
    get environment() {
        return this.editableEnvironment;
    }
    @Input() project: Project;

    @Output() deletedEnv = new EventEmitter<string>();

    constructor(
        private _toast: ToastService,
        private _router: Router,
        private _translate: TranslateService,
        private _authenticationStore: AuthentificationStore,
        private store: Store
    ) {
        this.currentUser = this._authenticationStore.getUser();
    }

    fetchUsage() {
        if (!this.project) {
            return;
        }
        this.loadingUsage = true;
        this.store.dispatch(new projectActions.FetchEnvironmentUsageInProject({
            projectKey: this.project.key,
            environmentName: this.environment.name
        })).pipe(finalize(() => this.loadingUsage = false))
            .subscribe();
    }

    ngOnInit() {
        this.fetchUsage();
    }

    renameEnvironment(): void {
        this.loading = true;
        this.store.dispatch(new projectActions.UpdateEnvironmentInProject({
            projectKey: this.project.key,
            environmentName: this.oldEnvName,
            changes: this.editableEnvironment
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('environment_renamed')));
    }

    cloneEnvironment(cloneModal?: any): void {
        this.cloneLoading = true;
        this.store.dispatch(new projectActions.CloneEnvironmentInProject({
            projectKey: this.project.key,
            cloneName: this.cloneName,
            environment: this.editableEnvironment
        })).pipe(finalize(() => {
            this.cloneLoading = false;
            this.cloneName = '';
            cloneModal.hide();
        })).subscribe(() => {
            this._toast.success('', this._translate.instant('environment_cloned'));
            this._router.navigate(['/project/', this.project.key], { queryParams: { tab: 'environments', envName: this.cloneName } });
        });
    }

    deleteEnvironment(): void {
        this.loading = true;
        this.store.dispatch(new projectActions.DeleteEnvironmentInProject({
            projectKey: this.project.key, environment: this.editableEnvironment
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('environment_deleted'));
                this.deletedEnv.emit(this.editableEnvironment.name);
            });
    }


    variableEvent(event: VariableEvent): void {
        event.variable.value = String(event.variable.value);
        switch (event.type) {
            case 'add':
                this.addVarLoading = true;
                this.store.dispatch(new projectActions.AddEnvironmentVariableInProject({
                    projectKey: this.project.key,
                    environmentName: this.editableEnvironment.name,
                    variable: event.variable
                })).pipe(finalize(() => this.addVarLoading = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_added')));
                break;
            case 'update':
                this.store.dispatch(new projectActions.UpdateEnvironmentVariableInProject({
                    projectKey: this.project.key,
                    environmentName: this.editableEnvironment.name,
                    variableName: event.variable.name,
                    changes: event.variable
                })).pipe(finalize(() => event.variable.updating = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_updated')));
                break;
            case 'delete':
                this.store.dispatch(new projectActions.DeleteEnvironmentVariableInProject({
                    projectKey: this.project.key,
                    environmentName: this.editableEnvironment.name,
                    variable: event.variable
                })).pipe(finalize(() => event.variable.updating = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_deleted')));
                break;
        }
    }
}
