import {Component, OnInit, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {ProjectStore} from '../../../service/project/project.store';
import {Project, LoadOpts} from '../../../model/project.model';
import {ToastService} from '../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {Subscription} from 'rxjs/Subscription';
import {PermissionValue} from '../../../model/permission.model';
import {User} from '../../../model/user.model';

@Component({
    selector: 'app-project-show',
    templateUrl: './project.html',
    styleUrls: ['./project.scss']
})
export class ProjectShowComponent implements OnInit, OnDestroy {
    currentUser: User;

    project: Project;
    private projectSubscriber: Subscription;

    selectedTab = 'applications';

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
            let goToDefaultTab = true;
            if (params['tab']) {
                this.selectedTab = params['tab'];
                goToDefaultTab = false;
            }
            this._route.params.subscribe(routeParams => {
                const key = routeParams['key'];
                if (key) {
                    if (this.project && this.project.key !== key) {
                        this.project = undefined;
                    }
                    if (!this.project) {
                        this.refreshDatas(key, goToDefaultTab);
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

    refreshDatas(key: string, goToDefaultTab: boolean): void {
        if (this.projectSubscriber) {
            this.projectSubscriber.unsubscribe();
        }
        let opts = [
          new LoadOpts('withApplicationNames', 'application_names'),
          new LoadOpts('withPipelineNames', 'pipeline_names'),
          new LoadOpts('withWorkflowNames', 'workflow_names'),
        ];

        if (this.selectedTab === 'variables') {
            opts.push(new LoadOpts('withVariables', 'variables'));
        } else if (this.selectedTab === 'environments') {
            opts.push(new LoadOpts('withEnvironments', 'environments'));
        } else if (this.selectedTab === 'permissions') {
            opts.push(new LoadOpts('withEnvironments', 'environments'));
        }

        this.projectSubscriber = this._projectStore.getProjects(key, opts).subscribe(prjs => {
            let proj = prjs.get(key);
            if (proj) {
                if (!proj.externalChange) {
                    this.project = proj;
                    if (goToDefaultTab) {
                        if (this.project.workflow_migration !== 'NOT_BEGUN') {
                            this.selectedTab = 'workflows';
                        } else {
                            this.selectedTab = 'applications';
                        }
                    }
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
}
