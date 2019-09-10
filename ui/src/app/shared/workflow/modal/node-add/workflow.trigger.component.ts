import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { ProjectIntegration } from 'app/model/integration.model';
import { Pipeline, PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, WNodeTrigger, Workflow, WorkflowNodeCondition, WorkflowNodeConditions } from 'app/model/workflow.model';
import { ApplicationService } from 'app/service/application/application.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { WorkflowNodeAddWizardComponent } from 'app/shared/workflow/wizard/node-add/node.wizard.component';
import { WorkflowWizardOutgoingHookComponent } from 'app/shared/workflow/wizard/outgoinghook/wizard.outgoinghook.component';
import cloneDeep from 'lodash-es/cloneDeep';
import { forkJoin, Observable } from 'rxjs';

@Component({
    selector: 'app-workflow-trigger',
    templateUrl: './workflow.trigger.html',
    styleUrls: ['./workflow.trigger.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowTriggerComponent {

    @ViewChild('triggerModal', { static: false })
    triggerModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    @ViewChild('nodeWizard', { static: false })
    nodeWizard: WorkflowNodeAddWizardComponent;
    @ViewChild('worklflowAddOutgoingHook', { static: false })
    worklflowAddOutgoingHook: WorkflowWizardOutgoingHookComponent;

    @Output() triggerEvent = new EventEmitter<Workflow>();
    @Input() source: WNode;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() loading: boolean;
    @Input() destination: string;
    @Input() editMode: boolean;

    destNode: WNode;
    currentSection = 'pipeline';
    selectedType: string;
    isParent: boolean;

    constructor(private _modalService: SuiModalService, private _pipService: PipelineService,
                private _envService: EnvironmentService, private _appService: ApplicationService) { }

    show(t: string, isP: boolean): void {
        this.selectedType = t;
        this.isParent = isP;
        const config = new TemplateModalConfig<boolean, boolean, void>(this.triggerModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
    }

    hide(): void {
        this.modal.approve(true);
    }

    destNodeChange(node: WNode): void {
        this.destNode = node;
    }

    pipelineSectionChanged(pipSection: string) {
        this.currentSection = pipSection;
    }

    addOutgoingHook(): void {
        this.destNode = this.worklflowAddOutgoingHook.hook;
        this.saveTrigger();
    }

    saveTrigger(): void {
        this.destNode.context.conditions = new WorkflowNodeConditions();
        this.destNode.context.conditions.plain = new Array<WorkflowNodeCondition>();
        let c = new WorkflowNodeCondition();
        c.variable = 'cds.status';
        c.value = PipelineStatus.SUCCESS;
        c.operator = 'eq';
        this.destNode.context.conditions.plain.push(c);
        if (this.editMode) {
            switch (this.destNode.type) {
                case 'pipeline':
                    this.destNode.name =
                        this.project.pipeline_names.find(p => p.id === this.destNode.context.pipeline_id).name;
                    break;
                case 'outgoinghook':
                    this.destNode.name = 'Outgoing';
                    break;
            }
            this.destNode.ref = new Date().getTime().toString();
        }
        let clonedWorkflow = cloneDeep(this.workflow);

        if (this.source && !this.isParent) {
            let sourceNode: WNode;
            if (!this.editMode) {
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
            return
        }
        if (this.editMode) {
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
                this.triggerEvent.emit(clonedWorkflow)
            });
        } else {
            this.triggerEvent.emit(clonedWorkflow);
        }

    }

    nextStep() {
        this.nodeWizard.goToNextSection().subscribe((section) => {
            if (section === 'done') {
                this.saveTrigger();
            } else {
                this.currentSection = section;
            }
        });
    }

    getApplication(w: Workflow): Observable<Application> {
        if (this.destNode.context.application_id) {
            if (w.applications && w.applications[this.destNode.context.application_id]) {
                return Observable.of(w.applications[this.destNode.context.application_id]);
            }
            return this._appService
                .getApplication(this.project.key, this.project.application_names
                    .find(a => a.id === this.destNode.context.application_id).name)
        }
        return Observable.of(null);
    }

    getPipeline(w: Workflow): Observable<Pipeline> {
        if (this.destNode.context.pipeline_id) {
            if (w.pipelines && w.pipelines[this.destNode.context.pipeline_id]) {
                return Observable.of(w.pipelines[this.destNode.context.pipeline_id]);
            }
            return this._pipService.getPipeline(this.project.key, this.project.pipeline_names
                .find(p => p.id === this.destNode.context.pipeline_id).name)
        }
        return Observable.of(null);
    }

    getEnvironment(w: Workflow): Observable<Environment> {
        if (this.destNode.context.environment_id) {
            if (w.environments && w.environments[this.destNode.context.environment_id]) {
                return Observable.of(w.environments[this.destNode.context.environment_id]);
            }
            return this._envService.getEnvironment(this.project.key, this.project.environment_names
                .find(e => e.id === this.destNode.context.environment_id).name);
        }
        return Observable.of(null);
    }

    getProjectIntegration(w: Workflow): Observable<ProjectIntegration> {
        if (this.destNode.context.project_integration_id) {
            return Observable.of(this.project.integrations.find(i => i.id === this.destNode.context.project_integration_id));
        }
        return Observable.of(null);
    }
}
