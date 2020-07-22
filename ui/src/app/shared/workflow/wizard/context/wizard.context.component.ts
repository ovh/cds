import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnDestroy,
    OnInit,
    Output
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { IdName, Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { ApplicationService } from 'app/service/application/application.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { FetchApplication } from 'app/store/applications.action';
import { ApplicationsState, ApplicationStateModel } from 'app/store/applications.state';
import { ProjectState } from 'app/store/project.state';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable, Subscription } from 'rxjs';
import { filter, finalize, first, flatMap } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-context',
    templateUrl: './wizard.context.html',
    styleUrls: ['./wizard.context.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWizardNodeContextComponent implements OnInit, OnDestroy {

    @Input() workflow: Workflow;
    @Input() readonly = true;
    @Output() contextChange = new EventEmitter<boolean>();

    project: Project;
    editMode: boolean;

    @Select(WorkflowState.getSelectedNode()) node$: Observable<WNode>;
    node: WNode;
    nodeSub: Subscription;

    environments: Environment[];
    applications: IdName[];
    integrations: Array<IdName>;
    loading: boolean;
    showCheckStatus = false;

    constructor(private _store: Store, private _appService: ApplicationService, private _translate: TranslateService,
        private _toast: ToastService, private _cd: ChangeDetectorRef) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this.editMode = this._store.selectSnapshot(WorkflowState).editMode;
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.nodeSub = this.node$.subscribe(n => {
           this.node = cloneDeep(n);
            if (this.node.context.application_id !== 0 && this.applications) {
                this.change();
            }
            this.updateVCSStatusCheck();
            this._cd.markForCheck();
        });

        let voidEnv = new Environment();
        voidEnv.id = 0;
        voidEnv.name = ' ';
        this.environments = cloneDeep(this.project.environments) || [];
        this.environments.unshift(voidEnv);

        let voidApp = new IdName();
        voidApp.id = 0;
        voidApp.name = ' ';
        this.applications = cloneDeep(this.project.application_names) || [];
        this.applications.unshift(voidApp);
        this.updateVCSStatusCheck();
        if (this.node.context.application_id !== 0) {
            this.change();
        }
    }

    pushChange(): void {
        this.contextChange.emit(true);
    }

    updateVCSStatusCheck(b?: boolean): void {
        if (!this.applications || !this.node) {
            return;
        }
        if (!this.node.context.application_id) {
            this.showCheckStatus = false;
            return;
        }
        if (b) {
            this.pushChange();
        }
        let i = this.applications.findIndex(a => a.id === this.node.context.application_id);
        if (i === -1) {
            this.showCheckStatus = false;
            return;
        }
        this._store.dispatch(new FetchApplication({ projectKey: this.project.key, applicationName: this.applications[i].name }))
            .pipe(
                flatMap(() => this._store.selectOnce(ApplicationsState.currentState())),
                filter((s: ApplicationStateModel) => s.application != null && s.application.name === this.applications[i].name),
                first())
            .subscribe(app => {
                this.showCheckStatus = app.application.repository_fullname && app.application.repository_fullname !== '';
            });
    }

    initIntegrationList(): void {
        let voidPF = new IdName();
        voidPF.id = 0;
        voidPF.name = '';
        this.integrations.unshift(voidPF);
    }

    change(): void {
        this.node.context.application_id = Number(this.node.context.application_id) || 0;
        this.node.context.environment_id = Number(this.node.context.environment_id) || 0;
        this.node.context.pipeline_id = Number(this.node.context.pipeline_id) || 0;

        let appName = '';
        if (this.node.context.application_id !== 0) {
            appName = this.applications.find(k => Number(k.id) === this.node.context.application_id).name;
        }
        if (appName && appName !== ' ') {
            this._appService.getDeploymentStrategies(this.project.key, appName).pipe(
                first(),
                finalize(() => this.initIntegrationList())
            ).subscribe(
                data => {
                    this.integrations = [];
                    let pfNames = Object.keys(data);
                    pfNames.forEach(s => {
                        let pf = this.project.integrations.find(p => p.name === s);
                        if (pf) {
                            let idName = new IdName();
                            idName.id = pf.id;
                            idName.name = pf.name;
                            this.integrations.push(idName);
                        }
                    });
                    let index = this.integrations
                        .findIndex(idName => idName.id === this.node.context.project_integration_id);
                    if (this.node.context.project_integration_id > 0 && index === -1) {
                        delete this.node.context.project_integration_id;
                    }
                    this._cd.markForCheck();
                }
            )
        } else {
            this.integrations = [];
            this.initIntegrationList();
            this.node.context.project_integration_id = 0;
        }
    }

    updateWorkflow(): void {
        this.loading = true;
        let clonedWorkflow = cloneDeep(this.workflow);
        let n: WNode;
        if (this.editMode) {
            n = Workflow.getNodeByRef(this.node.ref, clonedWorkflow);
        } else {
            n = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        }
        n.context.application_id = this.node.context.application_id;
        n.context.environment_id = this.node.context.environment_id;
        n.context.project_integration_id = this.node.context.project_integration_id;
        n.context.mutex = this.node.context.mutex;

        let previousName = n.name;
        n.name = this.node.name;

        if (previousName !== n.name) {
            if (clonedWorkflow.notifications) {
                clonedWorkflow.notifications.forEach(notif => {
                    if (notif.source_node_ref) {
                        notif.source_node_ref = notif.source_node_ref.map(ref => {
                           if (ref === previousName) {
                               return n.name;
                           }
                           return ref;
                        });
                    }

                });
            }
        }

        this._store.dispatch(new UpdateWorkflow({
            projectKey: this.workflow.project_key,
            workflowName: this.workflow.name,
            changes: clonedWorkflow
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this.contextChange.emit(false);
                if (this.editMode) {
                    this._toast.info('', this._translate.instant('workflow_ascode_updated'));
                } else {
                    this._toast.success('', this._translate.instant('workflow_updated'));
                }
            });
    }
}
