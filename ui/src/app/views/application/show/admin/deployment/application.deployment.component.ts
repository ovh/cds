import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddApplicationDeployment, DeleteApplicationDeployment, UpdateApplicationDeployment } from 'app/store/applications.action';
import { finalize } from 'rxjs/operators';
import { Application } from '../../../../../model/application.model';
import { ProjectIntegration } from '../../../../../model/integration.model';
import { Project } from '../../../../../model/project.model';
import { WarningModalComponent } from '../../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-application-deployment',
    templateUrl: './application.deployment.html',
    styleUrls: ['./application.deployment.scss']
})
export class ApplicationDeploymentComponent {

    filteredIntegrations: Array<ProjectIntegration>;
    selectedIntegration: ProjectIntegration;

    public loadingBtn = false;

    _project: Project;

    @ViewChild('removeWarning', {static: true}) removeWarningModal: WarningModalComponent;
    @ViewChild('linkWarning', {static: true}) linkWarningModal: WarningModalComponent;

    @Input() application: Application;
    @Input('project')
    set project(project: Project) {
        this._project = project;
        if (project.integrations) {
            this.filteredIntegrations = project.integrations.filter(p => p.model.deployment);
        }
    }
    get project(): Project {
        return this._project;
    }

    getIntegrationNames(): Array<string> {
        if (this.application.deployment_strategies) {
            return Object.keys(this.application.deployment_strategies);
        }
        return null;
    }

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private store: Store
    ) {

    }

    clickDeleteIntegration(pfName: string) {
        this.loadingBtn = true;
        this.store.dispatch(new DeleteApplicationDeployment({
            projectKey: this.project.key,
            applicationName: this.application.name,
            integrationName: pfName
        })).pipe(finalize(() => this.loadingBtn = false))
            .subscribe(() => this._toast.success('', this._translate.instant('application_integration_deleted')));
    }

    updateIntegration(pfName: string) {
        this.loadingBtn = true;
        this.store.dispatch(new UpdateApplicationDeployment({
            projectKey: this.project.key,
            applicationName: this.application.name,
            deploymentName: pfName,
            config: this.application.deployment_strategies[pfName]
        })).pipe(finalize(() => this.loadingBtn = false))
            .subscribe(() => this._toast.success('', this._translate.instant('application_integration_updated')));
    }

    addIntegration() {
        this.loadingBtn = true;
        if (this.selectedIntegration.model) {
            this.store.dispatch(new AddApplicationDeployment({
                projectKey: this.project.key,
                applicationName: this.application.name,
                integration: this.selectedIntegration
            })).pipe(finalize(() => this.loadingBtn = false))
                .subscribe(() => {
                    this.selectedIntegration = null;
                    this._toast.success('', this._translate.instant('application_integration_added'));
                });
        }
    }
}
