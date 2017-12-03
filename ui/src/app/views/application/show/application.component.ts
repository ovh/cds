import {Component, OnInit, ViewChild, QueryList, ViewChildren, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Application, ApplicationFilter} from '../../../model/application.model';
import {ApplicationStore} from '../../../service/application/application.store';
import {ProjectStore} from '../../../service/project/project.store';
import {Project} from '../../../model/project.model';
import {User} from '../../../model/user.model';
import {Pipeline} from '../../../model/pipeline.model';
import {Workflow} from '../../../model/workflow.model';
import {Environment} from '../../../model/environment.model';
import {environment} from '../../../../environments/environment';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {ApplicationWorkflowComponent} from './workflow/application.workflow.component';
import {VariableEvent} from '../../../shared/variable/variable.event.model';
import {ToastService} from '../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {Subscription} from 'rxjs/Subscription';
import {WarningModalComponent} from '../../../shared/modal/warning/warning.component';
import {CDSWorker} from '../../../shared/worker/worker';
import {NotificationEvent} from './notifications/notification.event';
import {ApplicationNotificationListComponent} from './notifications/list/notification.list.component';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-application-show',
    templateUrl: './application.html',
    styleUrls: ['./application.scss']
})
@AutoUnsubscribe()
export class ApplicationShowComponent implements OnInit, OnDestroy {

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
    workerSubscription: Subscription;
    worker: CDSWorker;

    // Selected tab
    selectedTab = 'workflow';

    @ViewChildren(ApplicationWorkflowComponent)
    workflowComponentList: QueryList<ApplicationWorkflowComponent>;

    @ViewChild('varWarning')
    private varWarningModal: WarningModalComponent;
    @ViewChild('permWarning')
    private permWarningModal: WarningModalComponent;
    @ViewChild('notifWarning')
    private notifWarningModal: WarningModalComponent;

    @ViewChild('notificationList')
    private notificationListComponent: ApplicationNotificationListComponent;

    // Filter
    appFilter: ApplicationFilter = {
        remote: '',
        branch: '',
        version: ' '
    };

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

    constructor(private _applicationStore: ApplicationStore, private _route: ActivatedRoute,
                private _router: Router, private _authStore: AuthentificationStore,
                private _toast: ToastService, public _translate: TranslateService,
                private _projectStore: ProjectStore) {
        this.currentUser = this._authStore.getUser();
        // Update data if route change
        this._route.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this._route.queryParams.subscribe(queryParams => {
           this.appFilter = {
               remote: queryParams['remote'] || '',
               branch: queryParams['branch'] || '',
               version: queryParams['version'] || ' '
           };

           if (this.project && this.application) {
               this.startWorker(this.project.key);
           }
        });

        if (this._route.snapshot && this._route.queryParams) {
            this.workflowName = this._route.snapshot.queryParams['workflow'];
            this.workflowNum = this._route.snapshot.queryParams['run'];
            this.workflowNodeRun = this._route.snapshot.queryParams['node'];
        }
        this.workflowPipeline = this._route.snapshot.queryParams['wpipeline'];
        this._route.params.subscribe(params => {
            let key = params['key'];
            let appName = params['appName'];
            if (key && appName) {
                if (this.applicationSubscription) {
                    this.applicationSubscription.unsubscribe();
                }
                if (this.application && this.application.name !== appName) {
                    this.application = null;
                    this.stopWorker();
                }
                if (!this.application) {
                    this.applicationSubscription = this._applicationStore.getApplications(key, appName, this.appFilter).subscribe(apps => {
                        if (apps) {
                            let updatedApplication = apps.get(key + '-' + appName);
                            if (updatedApplication && !updatedApplication.externalChange) {
                                this.readyApp = true;
                                this.application = updatedApplication;

                                this.workflows = updatedApplication.usage.workflows || [];
                                this.environments = updatedApplication.usage.environments || [];
                                this.pipelines = updatedApplication.usage.pipelines || [];
                                this.usageCount = this.pipelines.length + this.environments.length + this.workflows.length;

                                // Start worker
                                this.startWorker(key);

                                // Update recent application viewed
                                this._applicationStore.updateRecentApplication(key, this.application);

                                // Switch workflow
                                if (this.workflowComponentList && this.workflowComponentList.length > 0) {
                                    this.workflowComponentList.first.switchApplication(this.application);
                                }
                            } else if (updatedApplication && updatedApplication.externalChange) {
                                this._toast.info('', this._translate.instant('warning_application'));
                            }
                        }
                    }, () => {
                        this._router.navigate(['/project', key]);
                    });
                }
            }
        });
    }

    ngOnDestroy(): void {
        this.appFilter.remote = '';
        this.stopWorker();
    }

    ngOnInit() {
        this._route.queryParams.subscribe(params => {
            let tab = params['tab'];
            if (tab) {
                this.selectedTab = tab;
            }
        });
    }

    stopWorker(): void {
       if (this.worker) {
           this.worker.stop();
       }
    }

    /**
     * Start workers to pull workflow.
     */
    startWorker(key: string): void {
        this.stopWorker();

        if (this.application.workflows && this.application.workflows.length > 0) {
            let msgToSend = {
                'user': this.currentUser,
                'session': this._authStore.getSessionToken(),
                'api': environment.apiURL,
                'key': key,
                'appName': this.application.name,
                'branch': this.appFilter.branch || 'master',
                'remote': this.appFilter.remote,
                'version': this.appFilter.version
            };

            this.worker = new CDSWorker('assets/worker/web/workflow.js?appName=' + this.application.name);
            this.workerSubscription = this.worker.response().subscribe(msg => {
                if (this.application.workflows && this.workflowComponentList
                    && this.workflowComponentList.length > 0 && msg && msg !== '') {
                    this.workflowComponentList.first.refreshWorkflow(JSON.parse(msg));
                }
            });
            this.worker.start(msgToSend);
        }
    }

    /**
     * Reinit worker the the current tab.
     *
     */
    changeWorkerFilter(destroy: boolean): void {
        this.stopWorker();
        if (!destroy) {
            this.startWorker(this.project.key);
        }
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

    notificationEvent(event: NotificationEvent, skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.notifWarningModal.show(event);
        } else {
            switch (event.type) {
                case 'add':
                    this.notifFormLoading = true;
                    this._applicationStore.addNotifications(this.project.key, this.application.name, event.notifications).subscribe(() => {
                        this._toast.success('', this._translate.instant('notifications_added'));
                        if (this.notificationListComponent) {
                            this.notificationListComponent.close();
                        }
                        this.notifFormLoading = false;
                    }, () => {
                        this.notifFormLoading = false;
                    });
                    break;
                case 'update':
                    this.notifFormLoading = true;
                    this._applicationStore.updateNotification(
                        this.project.key,
                        this.application.name,
                        event.notifications[0].pipeline.name,
                        event.notifications[0]
                    ).subscribe(() => {
                        if (this.notificationListComponent) {
                            this.notificationListComponent.close();
                        }
                        this.notifFormLoading = false;
                        this._toast.success('', this._translate.instant('notification_updated'));
                    }, () => {
                        this.notifFormLoading = false;
                    });
                    break;
                case 'delete':
                    this.notifFormLoading = true;
                    this._applicationStore.deleteNotification(
                        this.project.key,
                        this.application.name,
                        event.notifications[0].pipeline.name,
                        event.notifications[0].environment.name
                    ).subscribe(() => {
                        if (this.notificationListComponent) {
                            this.notificationListComponent.close();
                        }
                        this.notifFormLoading = false;
                        this._toast.success('', this._translate.instant('notifications_deleted'));
                    }, () => {
                        this.notifFormLoading = false;
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
