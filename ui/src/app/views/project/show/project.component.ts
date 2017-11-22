import {Component, OnInit, ViewChild, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {ProjectStore, LoadOpts} from '../../../service/project/project.store';
import {Project} from '../../../model/project.model';
import {VariableEvent} from '../../../shared/variable/variable.event.model';
import {ToastService} from '../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {EnvironmentPermissionEvent, PermissionEvent} from '../../../shared/permission/permission.event.model';
import {Subscription} from 'rxjs/Subscription';
import {WarningModalComponent} from '../../../shared/modal/warning/warning.component';
import {PermissionValue} from '../../../model/permission.model';
import {Environment} from '../../../model/environment.model';
import {User} from '../../../model/user.model';

@Component({
    selector: 'app-project-show',
    templateUrl: './project.html',
    styleUrls: ['./project.scss']
})
export class ProjectShowComponent implements OnInit, OnDestroy {
    permFormLoading = false;
    permEnvFormLoading = false
    currentUser: User;

    project: Project;
    private projectSubscriber: Subscription;

    selectedTab = 'applications';

    @ViewChild('permWarning')
    permWarningModal: WarningModalComponent;
    @ViewChild('permEnvWarning')
    permEnvWarningModal: WarningModalComponent;
    @ViewChild('permEnvGroupWarning')
    permEnvGroupWarningModal: WarningModalComponent;

    permissionEnum = PermissionValue;

    // queryparam for breadcrum
    workflowName: string;
    workflowNum: string;
    workflowNodeRun: string;
    workflowPipeline: string;

    constructor(private _projectStore: ProjectStore, private _route: ActivatedRoute, private _router: Router,
                private _toast: ToastService, public _translate: TranslateService,
                private _authentificationStore: AuthentificationStore) {
        this.currentUser = this._authentificationStore.getUser();
    }

    ngOnDestroy(): void {
        if (this.projectSubscriber) {
            this.projectSubscriber.unsubscribe();
        }
    }

    ngOnInit() {
        this._route.queryParams.subscribe((params) => {
            if (params['tab']) {
                this.selectedTab = params['tab'];
            }
            this._route.params.subscribe(routeParams => {
                const key = routeParams['key'];
                if (key) {
                    if (this.project && this.project.key !== key) {
                        this.project = undefined;
                    }
                    if (!this.project) {
                        this.refreshDatas(key);
                    }
                }
            });
        });

        if (this._route.snapshot && this._route.snapshot.queryParams) {
            this.workflowName = this._route.snapshot.queryParams['workflow'];
            this.workflowNum = this._route.snapshot.queryParams['run'];
            this.workflowNodeRun = this._route.snapshot.queryParams['node'];
            this.workflowPipeline = this._route.snapshot.queryParams['wpipeline'];
        }
    }

    refreshDatas(key: string): void {
        if (this.projectSubscriber) {
            this.projectSubscriber.unsubscribe();
        }
        let opts = [
          new LoadOpts('withApplicationNames', 'application_names'),
          new LoadOpts('withPipelineNames', 'pipeline_names'),
          new LoadOpts('withWorkflowNames', 'workflow_names'),
          new LoadOpts('withApplicationPipelines', 'applications'),
        ];

        if (this.selectedTab === 'variables') {
            opts.push(new LoadOpts('withVariables', 'variables'));
        } else if (this.selectedTab === 'environments') {
            opts.push(new LoadOpts('withEnvironments', 'environments'));
        }


        this.projectSubscriber = this._projectStore.getProjectResolver(key, opts).subscribe(proj => {
            if (proj) {
                if (!proj.externalChange) {
                    this.project = proj;
                } else if (proj && proj.externalChange) {
                    if (this.project.externalChange) {
                        this._toast.info('', this._translate.instant('warning_project'));
                    }
                }
            }
        }, () => {
            this._router.navigate(['/home']);
        });
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '?tab=' + tab);
    }

    addEnvPermEvent(event: EnvironmentPermissionEvent, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.permEnvWarningModal.show(event);
        } else {
            this.permEnvFormLoading = true;
            this._projectStore.addEnvironmentPermission(this.project.key, event.env.name, event.gp).subscribe(() => {
                this._toast.success('', this._translate.instant('permission_added'));
                this.permEnvFormLoading = false;
            }, () => {
                this.permEnvFormLoading = false;
            });
        }
    }

    envGroupEvent(event: PermissionEvent, env: Environment, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            event.env = env;
            this.permEnvGroupWarningModal.show(event);
        } else {
            if (!env) {
                env = event.env;
            }
            switch (event.type) {
                case 'update':
                    this._projectStore.updateEnvironmentPermission(this.project.key, env.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_updated'));
                    });
                    break;
                case 'delete':
                    this._projectStore.removeEnvironmentPermission(this.project.key, env.name, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_deleted'));
                    });
                    break;
            }
        }
    }

    groupEvent(event: PermissionEvent, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.permWarningModal.show(event);
        } else {
            switch (event.type) {
                case 'add':
                    this.permFormLoading = true;
                    this._projectStore.addProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_added'));
                        this.permFormLoading = false;
                    }, () => {
                        this.permFormLoading = false;
                    });
                    break;
                case 'update':
                    this._projectStore.updateProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_updated'));
                    });
                    break;
                case 'delete':
                    this._projectStore.removeProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_deleted'));
                    });
                    break;
            }
        }
    }
}
