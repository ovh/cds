import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    inject,
    Input, OnInit,
    Output,
    ViewChild
} from '@angular/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { ProjectIntegration } from 'app/model/integration.model';
import { Pipeline, PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import {
    WNode,
    WNodeTrigger,
    WNodeType,
    Workflow,
    WorkflowNodeCondition,
    WorkflowNodeConditions
} from 'app/model/workflow.model';
import { ApplicationService } from 'app/service/application/application.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { WorkflowNodeAddWizardComponent } from 'app/shared/workflow/wizard/node-add/node.wizard.component';
import { WorkflowWizardOutgoingHookComponent } from 'app/shared/workflow/wizard/outgoinghook/wizard.outgoinghook.component';
import { WorkflowState } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { forkJoin, Observable, of } from 'rxjs';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { finalize } from 'rxjs/operators';
import { ToastService } from 'app/shared/toast/ToastService';
import {NZ_MODAL_DATA, NzModalRef} from 'ng-zorro-antd/modal';


interface IModalData {
    project: Project,
    workflow: Workflow,
    source: WNode,
    isParent: boolean,
    selectedType: string,
}

@Component({
    selector: 'app-workflow-trigger',
    templateUrl: './workflow.trigger.html',
    styleUrls: ['workflow.trigger.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowTriggerComponent implements OnInit {

    @ViewChild('nodeWizard') nodeWizard: WorkflowNodeAddWizardComponent;
    @ViewChild('worklflowAddOutgoingHook')
    worklflowAddOutgoingHook: WorkflowWizardOutgoingHookComponent;

    @Input() source: WNode;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() loading: boolean;
    @Input() destination: string;

    destNode: WNode;
    currentSection: number;
    selectedType: string;
    isParent: boolean;

    readonly nzModalData: IModalData = inject(NZ_MODAL_DATA);

    constructor(private _pipService: PipelineService, private _store: Store, private _toast: ToastService, private _modal: NzModalRef,
                private _envService: EnvironmentService, private _appService: ApplicationService, private _cd: ChangeDetectorRef) { }

    ngOnInit() {
        if (this.nzModalData) {
            this.project = this.nzModalData.project;
            this.workflow = this.nzModalData.workflow;
            this.isParent = this.nzModalData.isParent;
            this.source = this.nzModalData.source;
            this.selectedType = this.nzModalData.selectedType;
            this._cd.markForCheck();
        }
    }

    hide(): void {
        this._modal.close();
    }

    destNodeChange(node: WNode): void {
        this.destNode = node;
    }

    pipelineSectionChanged(pipSection: number) {
        this.currentSection = pipSection;
    }

    addOutgoingHook(): void {
        this.destNode = this.worklflowAddOutgoingHook.outgoingHook;
        this.saveTrigger();
    }

    saveTrigger(): void {
        this.destNode.context.conditions = new WorkflowNodeConditions();
        this.destNode.context.conditions.plain = new Array<WorkflowNodeCondition>();
        let c = new WorkflowNodeCondition();
        c.variable = 'cds.status';
        c.value = PipelineStatus.SUCCESS;
        c.operator = 'eq';
        let editMode = this._store.selectSnapshot(WorkflowState).editMode;
        this.destNode.context.conditions.plain.push(c);
        if (editMode) {
            let allNodes = Workflow.getAllNodes(this.workflow);
            this.destNode.ref = new Date().getTime().toString();

            if (this.destNode.type === WNodeType.PIPELINE) {
                    this.destNode.name =
                        this.project.pipeline_names.find(p => p.id === this.destNode.context.pipeline_id)
                            .name;
            }
            let nodeBaseName = this.destNode.name;
            let hasNodeToRename = true;
            let nameIndex = 1;
            do {
                if (allNodes.findIndex(
                    n => n.name === this.destNode.name && n.ref !== this.destNode.ref) === -1) {
                    hasNodeToRename = false;
                } else {
                    this.destNode.name = nodeBaseName + '_' + nameIndex;
                    nameIndex++;
                }
            } while (hasNodeToRename);
        }
        let clonedWorkflow = cloneDeep(this.workflow);

        if (this.source && !this.isParent) {
            let sourceNode: WNode;
            if (!editMode) {
                sourceNode = Workflow.getNodeByID(this.source.id, clonedWorkflow);
            } else {
                sourceNode = Workflow.getNodeByRef(this.source.ref, clonedWorkflow);
            }
            if (!sourceNode.triggers) {
                sourceNode.triggers = new Array<WNodeTrigger>();
            }
            let newTrigger = new WNodeTrigger();
            newTrigger.parent_node_name = sourceNode.ref;
            newTrigger.child_node = this.destNode;
            sourceNode.triggers.push(newTrigger);
        } else if (this.isParent) {
            this.destNode.triggers = new Array<WNodeTrigger>();
            let newTrigger = new WNodeTrigger();
            newTrigger.child_node = clonedWorkflow.workflow_data.node;
            this.destNode.triggers.push(newTrigger);
            this.destNode.context.default_payload = newTrigger.child_node.context.default_payload;
            newTrigger.child_node.context.default_payload = null;
            this.destNode.hooks = cloneDeep(clonedWorkflow.workflow_data.node.hooks);
            clonedWorkflow.workflow_data.node.hooks = [];
            clonedWorkflow.workflow_data.node = this.destNode;
        } else {
            return;
        }
        if (editMode) {
            forkJoin([
                this.getApplication(clonedWorkflow),
                this.getPipeline(clonedWorkflow),
                this.getEnvironment(clonedWorkflow),
                this.getProjectIntegration(clonedWorkflow)

            ]).subscribe(results => {
                let app = results[0];
                let pip = results[1];
                let env = results[2];
                let projIn = results[3];
                if (app) {
                    clonedWorkflow.applications[app.id] = app;
                }
                if (pip) {
                    clonedWorkflow.pipelines[pip.id] = pip;
                }
                if (env) {
                    clonedWorkflow.environments[env.id] = env;
                }
                if (projIn) {
                    clonedWorkflow.project_integrations[projIn.id] = projIn;
                }
                this.updateWorkflow(clonedWorkflow);
            });
        } else {
            this.updateWorkflow(clonedWorkflow);
        }

    }

    nextStep() {
        this.nodeWizard.goToNextSection().subscribe((section) => {
            if (section === 4) {
                this.saveTrigger();
            } else {
                this.currentSection = section;
            }
        });
    }

    getApplication(w: Workflow): Observable<Application> {
        if (this.destNode.context.application_id) {
            if (w.applications && w.applications[this.destNode.context.application_id]) {
                return of(w.applications[this.destNode.context.application_id]);
            }
            return this._appService
                .getApplication(this.project.key, this.project.application_names
                    .find(a => a.id === this.destNode.context.application_id).name);
        }
        return of(null);
    }

    getPipeline(w: Workflow): Observable<Pipeline> {
        if (this.destNode.context.pipeline_id) {
            if (w.pipelines && w.pipelines[this.destNode.context.pipeline_id]) {
                return of(w.pipelines[this.destNode.context.pipeline_id]);
            }
            return this._pipService.getPipeline(this.project.key, this.project.pipeline_names
                .find(p => p.id === this.destNode.context.pipeline_id).name);
        }
        return of(null);
    }

    getEnvironment(w: Workflow): Observable<Environment> {
        if (this.destNode.context.environment_id) {
            if (w.environments && w.environments[this.destNode.context.environment_id]) {
                return of(w.environments[this.destNode.context.environment_id]);
            }
            return this._envService.getEnvironment(this.project.key, this.project.environment_names
                .find(e => e.id === this.destNode.context.environment_id).name);
        }
        return of(null);
    }

    getProjectIntegration(w: Workflow): Observable<ProjectIntegration> {
        if (this.destNode.context.project_integration_id) {
            return of(this.project.integrations.find(i => i.id === this.destNode.context.project_integration_id));
        }
        return of(null);
    }

    updateWorkflow(w: Workflow): void {
        this.loading = true;
        let editMode = this._store.selectSnapshot(WorkflowState).editMode;
        this._store.dispatch(new UpdateWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            changes: w
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(() => {
            if (!editMode) {
                this._toast.success('', 'Workflow updated');
                this._modal.triggerOk().then();
            }
        }, () => {
            if (Array.isArray(this.source.hooks) && this.source.hooks.length) {
                this.source.hooks.pop();
            }
        });
    }
}
