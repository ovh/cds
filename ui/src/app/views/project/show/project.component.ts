import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import * as  immutable from 'immutable';
import {Subscription} from 'rxjs';
import {finalize} from 'rxjs/operators';
import {PermissionValue} from '../../../model/permission.model';
import {LoadOpts, Project} from '../../../model/project.model';
import {User} from '../../../model/user.model';
import {Warning} from '../../../model/warning.model';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {ProjectStore} from '../../../service/project/project.store';
import {WarningStore} from '../../../service/warning/warning.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-show',
    templateUrl: './project.html',
    styleUrls: ['./project.scss']
})
@AutoUnsubscribe()
export class ProjectShowComponent implements OnInit {
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
    loadingFav = false;

    allWarnings: Array<Warning>;
    warnVariable: Array<Warning>;
    warnPerm: Array<Warning>;
    warnKeys: Array<Warning>;
    warnVCS: Array<Warning>;
    warnApplications: Array<Warning>;
    warnPipelines: Array<Warning>;
    warnWorkflow: Array<Warning>;
    warnEnvironment: Array<Warning>;
    warningsSub: Subscription;

    constructor(private _projectStore: ProjectStore, private _route: ActivatedRoute, private _router: Router,
                private _toast: ToastService, public _translate: TranslateService,
                private _authentificationStore: AuthentificationStore, private _warningStore: WarningStore) {
        this.currentUser = this._authentificationStore.getUser();
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
                        if (this.project.workflow_migration !== 'NOT_BEGUN' && this.selectedTab === 'applications') {
                            this.selectedTab = 'workflows';
                            goToDefaultTab = false;
                        }
                    }
                }
            }
        }, () => {
            this._router.navigate(['/home']);
        });

        this.warningsSub = this._warningStore.getProjectWarnings(key).subscribe(ws => {
            this.splitWarnings(ws.get(key));
        });
    }

    splitWarnings(warnings: immutable.Map<string, Warning>): void {
        if (warnings) {
            this.allWarnings = warnings.toArray().sort((a, b) => {
                return a.id - b.id;
            });
            this.warnVariable = new Array<Warning>();
            this.warnPerm = new Array<Warning>();
            this.warnKeys = new Array<Warning>();
            this.warnVCS = new Array<Warning>();
            this.warnApplications = new Array<Warning>();
            this.warnPipelines = new Array<Warning>();
            this.warnWorkflow = new Array<Warning>();
            this.warnEnvironment = new Array<Warning>();
            warnings.valueSeq().toArray().forEach(v => {
                if (v.ignored) {
                    return;
                }
                if (v.application_name !== '') {
                    this.warnApplications.push(v);
                }
                if (v.pipeline_name !== '') {
                    this.warnPipelines.push(v);
                }
                if (v.environment_name !== '') {
                    this.warnEnvironment.push(v);
                }
                if (v.workflow_name !== '') {
                    this.warnWorkflow.push(v);
                }
                if (v.type.indexOf('_VARIABLE') !== -1) {
                    this.warnVariable.push(v);
                    return;
                }
                if (v.type.indexOf('_PERMISSION') !== -1) {
                    this.warnPerm.push(v);
                    return;
                }
                if (v.type.indexOf('_KEY') !== -1) {
                    this.warnKeys.push(v);
                    return;
                }
                if (v.type.indexOf('PROJECT_VCS') !== -1) {
                    this.warnVCS.push(v);
                    return;
                }
            });
        }
    }

    updateFav() {
      this.loadingFav = true;
      this._projectStore.updateFavorite(this.project.key)
        .pipe(finalize(() => this.loadingFav = false))
        .subscribe(() => this._toast.success('', this._translate.instant('common_favorites_updated')))
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '?tab=' + tab);
    }
}
