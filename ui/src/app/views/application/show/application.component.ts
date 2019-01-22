import { Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Application } from '../../../model/application.model';
import { Environment } from '../../../model/environment.model';
import { PermissionValue } from '../../../model/permission.model';
import { Pipeline } from '../../../model/pipeline.model';
import { Project } from '../../../model/project.model';
import { User } from '../../../model/user.model';
import { Workflow } from '../../../model/workflow.model';
import { ApplicationStore } from '../../../service/application/application.store';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { ProjectStore } from '../../../service/project/project.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from '../../../shared/modal/warning/warning.component';
import { PermissionEvent } from '../../../shared/permission/permission.event.model';
import { ToastService } from '../../../shared/toast/ToastService';
import { VariableEvent } from '../../../shared/variable/variable.event.model';
import { CDSWebWorker } from '../../../shared/worker/web.worker';

@Component({
    selector: 'app-application-show',
    templateUrl: './application.html',
    styleUrls: ['./application.scss']
})
@AutoUnsubscribe()
export class ApplicationShowComponent implements OnInit {

    // Flag to show the page or not
    public readyApp = false;
    public varFormLoading = false;
    public permFormLoading = false;
    public notifFormLoading = false;


    // Project & Application data
    project: Project;
    application: Application;

    // Subscription
    applicationSubscription: Subscription;
    projectSubscription: Subscription;
    workerSubscription: Subscription;
    _routeParamsSub: Subscription;
    _routeDataSub: Subscription;
    _queryParamsSub: Subscription;
    worker: CDSWebWorker;

    // Selected tab
    selectedTab = 'home';

    @ViewChild('varWarning')
    private varWarningModal: WarningModalComponent;
    @ViewChild('permWarning')
    private permWarningModal: WarningModalComponent;

    // queryparam for breadcrum
    workflowName: string;
    workflowNum: string;
    workflowNodeRun: string;
    workflowPipeline: string;

    pipelines: Array<Pipeline> = new Array<Pipeline>();
    workflows: Array<Workflow> = new Array<Workflow>();
    environments: Array<Environment> = new Array<Environment>();
    currentUser: User;
    usageCount = 0;


    perm = PermissionValue;

    constructor(private _applicationStore: ApplicationStore, private _route: ActivatedRoute,
                private _router: Router, private _authStore: AuthentificationStore,
                private _toast: ToastService, public _translate: TranslateService,
                private _projectStore: ProjectStore) {
        this.currentUser = this._authStore.getUser();
        // Update data if route change
        this._routeDataSub = this._route.data.subscribe(datas => {
            this.project = datas['project'];
        });

        if (this._route.snapshot && this._route.queryParams) {
            this.workflowName = this._route.snapshot.queryParams['workflow'];
            this.workflowNum = this._route.snapshot.queryParams['run'];
            this.workflowNodeRun = this._route.snapshot.queryParams['node'];
        }
        this.workflowPipeline = this._route.snapshot.queryParams['wpipeline'];
        this._routeParamsSub = this._route.params.subscribe(params => {
            let key = params['key'];
            let appName = params['appName'];
            if (key && appName) {
                if (this.applicationSubscription) {
                    this.applicationSubscription.unsubscribe();
                }
                if (this.application && this.application.name !== appName) {
                    this.application = null;
                }
                if (!this.application) {
                    this.applicationSubscription = this._applicationStore
                        .getApplications(key, appName).subscribe(apps => {
                        if (apps) {
                            let updatedApplication = apps.get(key + '-' + appName);
                            if (updatedApplication && !updatedApplication.externalChange) {
                                this.readyApp = true;
                                this.application = updatedApplication;
                                this.workflows = updatedApplication.usage.workflows || [];
                                this.environments = updatedApplication.usage.environments || [];
                                this.pipelines = updatedApplication.usage.pipelines || [];
                                this.usageCount = this.pipelines.length + this.environments.length + this.workflows.length;

                                // Update recent application viewed
                                this._applicationStore.updateRecentApplication(key, this.application);
                            }
                        }
                    }, () => {
                        this._router.navigate(['/project', key], {queryParams: {tab: 'applications'}});
                    });
                }
            }
        });
    }

    ngOnInit() {
       this._queryParamsSub = this._route.queryParams.subscribe(params => {
            let tab = params['tab'];
            if (tab) {
                this.selectedTab = tab;
            }
        });

        this.projectSubscription = this._projectStore.getProjects(this.project.key)
            .subscribe((proj) => {
                if (!this.project || !proj || !proj.get(this.project.key)) {
                    return;
                }
                this.project = proj.get(this.project.key);
            });
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '/application/' + this.application.name + '?tab=' + tab);
    }

    /**
     * Event on variable
     * @param event
     */
    variableEvent(event: VariableEvent, skip?: boolean) {
        if (!skip && this.application.externalChange) {
            this.varWarningModal.show(event);
        } else {
            event.variable.value = String(event.variable.value);
            switch (event.type) {
                case 'add':
                    this.varFormLoading = true;
                    this._applicationStore.addVariable(this.project.key, this.application.name, event.variable).pipe(finalize(() => {
                        event.variable.updating = false;
                        this.varFormLoading = false;
                    })).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_added'));
                    });
                    break;
                case 'update':
                    this._applicationStore.updateVariable(this.project.key, this.application.name, event.variable).pipe(finalize(() => {
                        event.variable.updating = false;
                    })).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_updated'));
                    });
                    break;
                case 'delete':
                    this._applicationStore.removeVariable(this.project.key, this.application.name, event.variable).pipe(finalize(() => {
                        event.variable.updating = false;
                    })).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_deleted'));
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
        if (!skip && this.application.externalChange) {
            this.permWarningModal.show(event);
        } else {
            event.gp.permission = Number(event.gp.permission);
            switch (event.type) {
                case 'add':
                    this.permFormLoading = true;
                    this._applicationStore.addPermission(this.project.key, this.application.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_added'));
                        this.permFormLoading = false;
                    }, () => {
                        this.permFormLoading = false;
                    });
                    break;
                case 'update':
                    this._applicationStore.updatePermission(this.project.key, this.application.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_updated'));
                    });
                    break;
                case 'delete':
                    this._applicationStore.removePermission(this.project.key, this.application.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_deleted'));
                    });
                    break;
            }
        }
    }
}
