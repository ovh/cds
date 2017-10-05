import {Component, OnInit, ViewChild, OnDestroy} from '@angular/core';
import {ActivatedRoute, Params, Router} from '@angular/router';
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

    // optionnal application data
    workflowName: string;
    application: Application;
    version: string;
    buildNumber: string;
    envName: string;
    branch: string;

    queryParams: Params;

    @ViewChild('permWarning')
        permissionModalWarning: WarningModalComponent;
    @ViewChild('paramWarning')
    parameterModalWarning: WarningModalComponent;

    // Selected tab
    selectedTab = 'pipeline';

    constructor(private _routeActivated: ActivatedRoute, private _pipStore: PipelineStore,
        private _router: Router, private _toast: ToastService, public _translate: TranslateService,
        private _appPipService: ApplicationPipelineService) {
        this.project = this._routeActivated.snapshot.data['project'];
        if (this._routeActivated.snapshot.data['application']) {
            this.application = this._routeActivated.snapshot.data['application'];
        }
        if (this._routeActivated.snapshot.queryParams['version']) {
            this.version = this._routeActivated.snapshot.queryParams['version'];
        }
        if (this._routeActivated.snapshot.queryParams['buildNumber']) {
            this.buildNumber = this._routeActivated.snapshot.queryParams['buildNumber'];
        }
        if (this._routeActivated.snapshot.queryParams['envName']) {
            this.envName = this._routeActivated.snapshot.queryParams['envName'];
        }
        if (this._routeActivated.snapshot.queryParams['branch']) {
            this.branch = this._routeActivated.snapshot.queryParams['branch'];
        }
        if (this._routeActivated.snapshot.queryParams['workflow']) {
            this.workflowName = this._routeActivated.snapshot.queryParams['workflow'];
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
                        this.applications = pipelineUpdated.attached_application || [];
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
                        .finally(() => this.paramFormLoading = false)
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
