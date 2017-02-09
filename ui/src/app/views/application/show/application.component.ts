import {Component, OnInit, ViewChild, QueryList, ViewChildren, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Application, ApplicationFilter} from '../../../model/application.model';
import {ApplicationStore} from '../../../service/application/application.store';
import {Project} from '../../../model/project.model';
import {environment} from '../../../../environments/environment';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {ApplicationWorkflowComponent} from './workflow/application.workflow.component';
import {VariableEvent} from '../../../shared/variable/variable.event.model';
import {ToastService} from '../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {Subscription} from 'rxjs/Subscription';
import {WarningModalComponent} from '../../../shared/modal/warning/warning.component';
import {WorkflowItem} from '../../../model/application.workflow.model';
import {CDSWorker} from '../../../shared/worker/worker';

@Component({
    selector: 'app-application-show',
    templateUrl: './application.html',
    styleUrls: ['./application.scss']
})
export class ApplicationShowComponent implements OnInit, OnDestroy {

    // Flag to show the page or not
    private readyApp = false;
    public varFormLoading = false;
    public permFormLoading = false;


    // Project & Application data
    project: Project;
    application: Application;

    // Init workflow
    workflowInit = false;

    // Subscription
    applicationSubscription: Subscription;
    workerSubscription: Subscription;

    // Selected tab
    selectedTab = 'workflow';

    // worker
    worker: CDSWorker;

    @ViewChildren(ApplicationWorkflowComponent)
    workflowComponentList: QueryList<ApplicationWorkflowComponent>;

    @ViewChild('varWarning')
    private varWarningModal: WarningModalComponent;
    @ViewChild('permWarning')
    private permWarningModal: WarningModalComponent;

    // Filter
    appFilter: ApplicationFilter = {
        branch: '',
        version: 0
    };

    constructor(private _applicationStore: ApplicationStore, private _route: ActivatedRoute,
                private _router: Router, private _authStore: AuthentificationStore,
                private _toast: ToastService, public _translate: TranslateService) {
        // Update data if route change
        this._route.data.subscribe( datas => {
            this.project = datas['project'];
        });

        this._route.params.subscribe(params => {
            let key = params['key'];
            let appName = params['appName'];
            if (key && appName) {
                 if (this.applicationSubscription) {
                    this.applicationSubscription.unsubscribe();
                }
                this.applicationSubscription = this._applicationStore.getApplications(key, appName).subscribe(apps => {
                    if (apps) {
                        let updatedApplication = apps.get(key + '-' + appName);
                        if (updatedApplication && !updatedApplication.externalChange) {
                            this.readyApp = true;
                            this.application = updatedApplication;
                            this.checkOtherAppInWorkflow(key);
                            this.startWorker(key, appName);
                            this._applicationStore.updateRecentApplication(key, this.application);
                            if (this.workflowComponentList && this.workflowComponentList.length > 0) {
                                this.workflowComponentList.first.switchApplication();
                            }
                        } else if (updatedApplication && updatedApplication.externalChange) {
                            // TODO show warning
                        }
                    }
                }, () => {
                    this._router.navigate(['/project', key]);
                });
            }
        });
    }

    ngOnDestroy(): void {
        if (this.applicationSubscription) {
            this.applicationSubscription.unsubscribe();
        }
        if (this.worker) {
            this.worker.updateWorker('unsubscribe', {});
        }
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
    }

    ngOnInit() {
        this._route.queryParams.subscribe(params => {
            let tab = params['tab'];
            if (tab) {
                this.selectedTab = tab;
            }
        });
    }

    /**
     * Start worker to pull workflow.
     * WebWorker for Safari and EDGE
     * SharedWorker for the others  (worker shared between tabs)
     */
    startWorker(key: string, appName: string): void {
        if (this.application.workflows && this.application.workflows.length > 0) {
            let msgToSend = {
                'user': this._authStore.getUser(),
                'api': environment.apiURL,
                'key': key,
                'appName': appName,
                'branch': this.appFilter.branch,
                'version': this.appFilter.version
            };


            if (!this.worker) {
                this.worker = new CDSWorker('./assets/worker/shared/workflow.js', './assets/worker/web/workflow.js');
            } else {
                this.worker.updateWorker('unsubscribe', {});
            }
            this.worker.start(msgToSend);

            if (this.worker.webWorkerId) {
                this.checkOtherAppInWorkflow(key);
            }

            if (!this.workerSubscription) {
                this.workerSubscription = this.worker.response().subscribe(msg => {
                    if (msg.worker_id && !msg.data) {
                        this.checkOtherAppInWorkflow(key);
                    }
                    if (this.application.workflows && this.workflowComponentList
                        && this.workflowComponentList.length > 0 && msg.data && msg.data !== '') {
                            this.workflowComponentList.first.refreshWorkflow(JSON.parse(msg.data));
                    }
                });
            }
        }
    }

    checkOtherAppInWorkflow(key: string): void {
        if (this.application && this.application.workflows) {
            this.workflowInit = true;
            this.application.workflows.forEach( w => {
                this.isAnOtherApp(w, key);
            });
        }

    }

    isAnOtherApp(w: WorkflowItem, key: string): void {
        if (this.application.id !== w.application.id) {
            this.worker.updateWorker('subscribe', {
                'user': this._authStore.getUser(),
                'api': environment.apiURL,
                'key': key,
                'appName': w.application.name,
                'branch': this.appFilter.branch,
                'version': this.appFilter.version
            });
        }
        if (w.subPipelines) {
            w.subPipelines.forEach( sub => {
                this.isAnOtherApp(sub, key);
            });
        }
    }

    /**
     * Reinit worker the the current tab.
     *
     */
    changeWorkerFilter(): void {
        this.worker.start({
            'user': this._authStore.getUser(),
            'api': environment.apiURL,
            'key': this.project.key,
            'appName': this.application.name,
            'branch': this.appFilter.branch,
            'version': this.appFilter.version
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
            switch (event.type) {
                case 'add':
                    this.varFormLoading = true;
                    this._applicationStore.addVariable(this.project.key, this.application.name, event.variable).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_added'));
                        this.varFormLoading = false;
                    }, () => {
                        this.varFormLoading = false;
                    });
                    break;
                case 'update':
                    this._applicationStore.updateVariable(this.project.key, this.application.name, event.variable).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_updated'));
                    });
                    break;
                case 'delete':
                    this._applicationStore.removeVariable(this.project.key, this.application.name, event.variable).subscribe(() => {
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
