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
import {NotificationEvent} from './notifications/notification.event';
import {ApplicationNotificationListComponent} from './notifications/list/notification.list.component';

@Component({
    selector: 'app-application-show',
    templateUrl: './application.html',
    styleUrls: ['./application.scss']
})
export class ApplicationShowComponent implements OnInit, OnDestroy {

    // Flag to show the page or not
    public readyApp = false;
    public varFormLoading = false;
    public permFormLoading = false;
    public notifFormLoading = false;


    // Project & Application data
    project: Project;
    application: Application;

    // Init workflow
    workflowInit = false;

    // Subscription
    applicationSubscription: Subscription;
    workersSubscription: Map<string, Subscription> = new Map<string, Subscription>();
    workers: Map<string, CDSWorker> = new Map<string, CDSWorker>();
    applicationsWorflow: Array<string> = new Array<string>();

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

        this._route.queryParams.subscribe(queryParams => {
           this.appFilter.branch = queryParams['branch'] ? queryParams['branch'] : 'master';
           this.appFilter.version = queryParams['version'] ? Number(queryParams['version']) : 0;
           if (this.project && this.application) {
               this.startWorkers(this.project.key);
           }
        });
        this._route.params.subscribe(params => {
            let key = params['key'];
            let appName = params['appName'];
            if (key && appName) {
                 if (this.applicationSubscription) {
                    this.applicationSubscription.unsubscribe();
                }
                if (this.application && this.application.name !== appName) {
                     this.application = undefined;
                    this.stopWorkers();
                }
                if (!this.application) {
                    this.applicationSubscription = this._applicationStore.getApplications(key, appName).subscribe(apps => {
                        if (apps) {
                            let updatedApplication = apps.get(key + '-' + appName);
                            if (updatedApplication && !updatedApplication.externalChange &&
                                (!this.application || this.application.last_modified < updatedApplication.last_modified)) {
                                this.readyApp = true;
                                this.application = updatedApplication;

                                // List applications in workflow
                                this.checkOtherAppInWorkflow(key);

                                // Start worker
                                this.startWorkers(key);

                                // Update recent application viewed
                                this._applicationStore.updateRecentApplication(key, this.application);

                                // Switch workflow
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
            }
        });
    }

    ngOnDestroy(): void {
        if (this.applicationSubscription) {
            this.applicationSubscription.unsubscribe();
        }
       this.stopWorkers();
    }

    ngOnInit() {
        this._route.queryParams.subscribe(params => {
            let tab = params['tab'];
            if (tab) {
                this.selectedTab = tab;
            }
        });
    }

    stopWorkers(): void {
        if (this.workers) {
            this.workers.forEach( v => {
                v.stop();
            });
        }
        if (this.workersSubscription) {
            this.workersSubscription.forEach( v => {
                v.unsubscribe();
            });
        }
        this.workersSubscription = new Map<string, Subscription>();
        this.workers = new Map<string, CDSWorker>();
    }

    /**
     * Start workers to pull workflow.
     */
    startWorkers(key: string): void {
        this.stopWorkers();
        this.checkOtherAppInWorkflow(key);

        if (this.application.workflows && this.application.workflows.length > 0) {
            let msgToSend = {
                'user': this._authStore.getUser(),
                'session': this._authStore.getSessionToken(),
                'api': environment.apiURL,
                'key': key,
                'appName': '',
                'branch': this.appFilter.branch,
                'version': this.appFilter.version
            };

            this.applicationsWorflow.forEach(a => {
                let cdsWorker = new CDSWorker('assets/worker/web/workflow.js?appName=' + a);

                let sub = cdsWorker.response().subscribe(msg => {
                    if (this.application.workflows && this.workflowComponentList
                        && this.workflowComponentList.length > 0 && msg && msg !== '') {
                        this.workflowComponentList.first.refreshWorkflow(JSON.parse(msg));
                    }
                });
                this.workersSubscription.set(a, sub);

                msgToSend.appName = a;
                cdsWorker.start(msgToSend);
                this.workers.set(a, cdsWorker);


            });
        }
    }

    /**
     * Browse all sub applications in workflow and add worker subscription
     * @param key
     */
    checkOtherAppInWorkflow(key: string): void {
        this.applicationsWorflow = new Array<string>();
        if (this.application && this.application.workflows) {
            this.workflowInit = true;
            this.application.workflows.forEach( w => {
                this.isAnOtherApp(w, key);
            });
        }
    }

    /**
     * Recursive function adding worker subscription on sub application
     * @param w
     * @param key
     */
    isAnOtherApp(w: WorkflowItem, key: string): void {
        if (!this.applicationsWorflow.find(a => {
            return a === w.application.name;
            })) {
            this.applicationsWorflow.push(w.application.name);
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
        this.stopWorkers();
        this.startWorkers(this.project.key);
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
