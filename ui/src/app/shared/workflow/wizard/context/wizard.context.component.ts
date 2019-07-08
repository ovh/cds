import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { PermissionValue } from 'app/model/permission.model';
import { IdName, Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { ApplicationService } from 'app/service/application/application.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { FetchApplication } from 'app/store/applications.action';
import { ApplicationsState } from 'app/store/applications.state';
import { UpdateWorkflow } from 'app/store/workflow.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { filter, finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-context',
    templateUrl: './wizard.context.html',
    styleUrls: ['./wizard.context.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWizardNodeContextComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;
    editableNode: WNode;
    @Input('node') set node(data: WNode) {
        if (data) {
            this.editableNode = cloneDeep(data);
            if (this.editableNode.context.application_id !== 0 && this.applications) {
                this.change();
            }
            this.updateVCSStatusCheck();
        }
    };
    get node(): WNode {
        return this.editableNode;
    }
    @Input() readonly = true;

    @Output() contextChange = new EventEmitter<boolean>();

    environments: Environment[];
    applications: IdName[];
    integrations: Array<IdName>;
    permissionEnum = PermissionValue;
    loading: boolean;
    showCheckStatus = false;

    constructor(private _store: Store, private _appService: ApplicationService, private _translate: TranslateService,
        private _toast: ToastService, private _cd: ChangeDetectorRef) {
    }

    ngOnInit() {
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
        if (this.editableNode.context.application_id !== 0) {
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
        this._store.dispatch(new FetchApplication({ projectKey: this.project.key, applicationName: this.applications[i].name }));
        this._store.selectOnce(ApplicationsState.selectApplication(this.project.key, this.applications[i].name))
            .pipe(filter((app) => app != null), first())
            .subscribe((app: Application) => {
                this.showCheckStatus = app.repository_fullname && app.repository_fullname !== '';
            })
    }

    initIntegrationList(): void {
        let voidPF = new IdName();
        voidPF.id = 0;
        voidPF.name = '';
        this.integrations.unshift(voidPF);
    }

    change(): void {
        this.node.context.application_id = Number(this.node.context.application_id);
        this.node.context.environment_id = Number(this.node.context.environment_id);
        this.node.context.pipeline_id = Number(this.node.context.pipeline_id);

        let appName = this.applications.find(k => Number(k.id) === this.node.context.application_id).name;
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
        let n = Workflow.getNodeByID(this.editableNode.id, clonedWorkflow);
        n.context.application_id = this.editableNode.context.application_id;
        n.context.environment_id = this.editableNode.context.environment_id;
        n.context.project_integration_id = this.editableNode.context.project_integration_id;
        n.context.mutex = this.editableNode.context.mutex;

        let previousName = n.name;
        n.name = this.editableNode.name;

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
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.contextChange.emit(false);
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }
}
