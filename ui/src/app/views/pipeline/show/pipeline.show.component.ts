import {Component, OnInit, ViewChild, OnDestroy} from '@angular/core';
import {ActivatedRoute, Params, Router} from '@angular/router';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {User} from '../../../model/user.model';
import {Workflow} from '../../../model/workflow.model';
import {Environment} from '../../../model/environment.model';
import {Pipeline} from '../../../model/pipeline.model';
import {Project} from '../../../model/project.model';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {Subscription} from 'rxjs/Subscription';
import {WarningModalComponent} from '../../../shared/modal/warning/warning.component';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {TranslateService} from 'ng2-translate/ng2-translate';
import {ToastService} from '../../../shared/toast/ToastService';
import {ParameterEvent} from '../../../shared/parameter/parameter.event.model';
import {Application} from '../../../model/application.model';
import {ApplicationPipelineService} from '../../../service/application/pipeline/application.pipeline.service';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-pipeline-show',
    templateUrl: './pipeline.show.html',
    styleUrls: ['./pipeline.show.scss']
})
export class PipelineShowComponent implements OnInit, OnDestroy {

    public permFormLoading = false;
    public paramFormLoading = false;

    project: Project;
    pipeline: Pipeline;
    pipelineSubscriber: Subscription;

    applications: Array<Application> = new Array<Application>();
    workflows: Array<Workflow> = new Array<Workflow>();
    environments: Array<Environment> = new Array<Environment>();
    currentUser: User;
    usageCount = 0;

    // optionnal application data
    workflowName: string;
    application: Application;
    version: string;
    buildNumber: string;
    envName: string;
    branch: string;
    remote: string;

    queryParams: Params;

    @ViewChild('permWarning')
        permissionModalWarning: WarningModalComponent;
    @ViewChild('paramWarning')
    parameterModalWarning: WarningModalComponent;

    // Selected tab
    selectedTab = 'pipeline';

    constructor(private _routeActivated: ActivatedRoute, private _pipStore: PipelineStore,
        private _router: Router, private _toast: ToastService, public _translate: TranslateService,
        private _appPipService: ApplicationPipelineService, private _authentificationStore: AuthentificationStore) {
        this.currentUser = this._authentificationStore.getUser();
        this.project = this._routeActivated.snapshot.data['project'];
        this.application = this._routeActivated.snapshot.data['application'];

        this.buildNumber = this.getQueryParam('buildNumber');
        this.version = this.getQueryParam('version');
        this.envName = this.getQueryParam('envName');
        this.branch = this.getQueryParam('branch');
        this.remote = this.getQueryParam('remote');
    }

    getQueryParam(name: string): string {
        if (this._routeActivated.snapshot.queryParams[name]) {
            return this._routeActivated.snapshot.queryParams[name];
        }
    }

    ngOnDestroy(): void {
        if (this.pipelineSubscriber) {
            this.pipelineSubscriber.unsubscribe();
        }
    }

    ngOnInit() {
        this._routeActivated.params.subscribe(params => {
            let key = params['key'];
            let pipName = params['pipName'];
            if (key && pipName) {
                this.refreshDatas(key, pipName);
            }
        });

        this._routeActivated.queryParams.subscribe(params => {
            this.queryParams = params;
            let tab = params['tab'] ;
            if (tab) {
                this.selectedTab = tab;
            }
        });
    }

    refreshDatas(key: string, pipName: string): void {
        if (this.pipelineSubscriber) {
            this.pipelineSubscriber.unsubscribe();
        }
        if (this.pipeline && this.pipeline.name !== pipName) {
            this.pipeline = undefined;
        }
        if (!this.pipeline) {
            this.pipelineSubscriber = this._pipStore.getPipelines(key, pipName).subscribe( pip => {
                if (pip) {
                    let pipelineUpdated = pip.get(key + '-' + pipName);
                    if (pipelineUpdated && !pipelineUpdated.externalChange &&
                        (!this.pipeline || this.pipeline.last_modified < pipelineUpdated.last_modified)) {
                        this.pipeline = pipelineUpdated;
                        this.applications = pipelineUpdated.usage.applications || [];
                        this.workflows = pipelineUpdated.usage.workflows || [];
                        this.environments = pipelineUpdated.usage.environments || [];
                        this.usageCount = this.applications.length + this.environments.length + this.workflows.length;
                    } else if (pipelineUpdated && pipelineUpdated.externalChange) {
                        this._toast.info('', this._translate.instant('warning_pipeline'));
                    }
                }
            }, () => {
                this._router.navigate(['/project', key]);
            });
        }
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '/pipeline/' + this.pipeline.name + '?tab=' + tab);
    }

    parameterEvent(event: ParameterEvent, skip?: boolean): void {
        if (!skip && this.pipeline.externalChange) {
            this.parameterModalWarning.show(event);
        } else {
            if (event.parameter) {
                event.parameter.value = String(event.parameter.value);
            }
            switch (event.type) {
                case 'add':
                    this.paramFormLoading = true;
                    this._pipStore.addParameter(this.project.key, this.pipeline.name, event.parameter)
                        .pipe(finalize(() => this.paramFormLoading = false))
                        .subscribe(() => this._toast.success('', this._translate.instant('parameter_added')));
                    break;
                case 'update':
                    this._pipStore.updateParameter(this.project.key, this.pipeline.name, event.parameter).subscribe(() => {
                        this._toast.success('', this._translate.instant('parameter_updated'));
                    });
                    break;
                case 'delete':
                    this._pipStore.removeParameter(this.project.key, this.pipeline.name, event.parameter).subscribe(() => {
                        this._toast.success('', this._translate.instant('parameter_deleted'));
                    });
                    break;
            }
        }
    }

    /**
     * Event on permission
     * @param event
     */
    groupEvent(event: PermissionEvent, skip?: boolean): void {
        if (!skip && this.pipeline.externalChange) {
            this.permissionModalWarning.show(event);
        } else {
            event.gp.permission = Number(event.gp.permission);
            switch (event.type) {
                case 'add':
                    this.permFormLoading = true;
                    this._pipStore.addPermission(this.project.key, this.pipeline.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_added'));
                        this.permFormLoading = false;
                    }, () => {
                        this.permFormLoading = false;
                    });
                    break;
                case 'update':
                    this._pipStore.updatePermission(this.project.key, this.pipeline.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_updated'));
                    });
                    break;
                case 'delete':
                    this._pipStore.removePermission(this.project.key, this.pipeline.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_deleted'));
                    });
                    break;
            }
        }
    }
}
