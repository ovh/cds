import {Component, Input, ViewChild} from '@angular/core';
import {Application} from '../../../../../model/application.model';
import {ProjectPlatform} from '../../../../../model/platform.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {Project} from '../../../../../model/project.model';
// import {ToastService} from '../../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {WarningModalComponent} from '../../../../../shared/modal/warning/warning.component';
import {finalize, first} from 'rxjs/operators';

@Component({
    selector: 'app-application-deployment',
    templateUrl: './application.deployment.html',
    styleUrls: ['./application.deployment.scss']
})
export class ApplicationDeploymentComponent {

    filteredPlatforms: Array<ProjectPlatform>;
    selectedPlatform: ProjectPlatform;

    public loadingBtn = false;

    _project: Project;

    @ViewChild('removeWarning') removeWarningModal: WarningModalComponent;
    @ViewChild('linkWarning') linkWarningModal: WarningModalComponent;

    @Input() application: Application;
    @Input('project')
    set project(project: Project) {
        this._project = project;
        if (project.platforms) {
            this.filteredPlatforms = project.platforms.filter(p => p.model.deployment);
        }
    }
    get project(): Project {
        return this._project;
    }

    getPlatformNames(): Array<string> {
        if (this.application.deployment_strategies) {
            return Object.keys(this.application.deployment_strategies);
        }
        return null;
    }

    clickDeletePlatform(pfName: string) {
        this.loadingBtn = true;
        this._appStore.deleteDeploymentStrategy(
            this._project.key,
            this.application.name,
            pfName)
        .pipe(
            first(),
            finalize(() => this.loadingBtn = false)
        ).subscribe(
            app => {
                this.application = app;
            }
        );
    }

    clickSavePlatform(pfName: string) {
        this.loadingBtn = true;
        this._appStore.saveDeploymentStrategy(
            this._project.key,
            this.application.name,
            pfName,
            this.application.deployment_strategies[pfName])
        .pipe(
            first(),
            finalize(() => this.loadingBtn = false)
        ).subscribe(
            app => {
                this.application = app;
            }
        );
    }

    savePlatform() {
        this.loadingBtn = true;
        if (this.selectedPlatform.model) {
            this._appStore.saveDeploymentStrategy(
                this._project.key,
                this.application.name,
                this.selectedPlatform.name,
                this.selectedPlatform.model.deployment_default_config)
            .pipe(
                first(),
                finalize(() => this.loadingBtn = false))
            .subscribe(
                app => {
                    this.application = app;
                    this.selectedPlatform = null;
                }
            );
        }
    }

    constructor(private _appStore: ApplicationStore,
                /*private _toast: ToastService,*/ public _translate: TranslateService) {
    }

}
