import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { FetchPipeline } from 'app/store/pipelines.action';
import { PipelinesState } from 'app/store/pipelines.state';
import { cloneDeep } from 'lodash';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { Subscription } from 'rxjs';
import { finalize, flatMap } from 'rxjs/operators';
import { Application } from '../../../../model/application.model';
import { PermissionValue } from '../../../../model/permission.model';
import { Pipeline } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { WNode, Workflow } from '../../../../model/workflow.model';
import { ApplicationWorkflowService } from '../../../../service/application/application.workflow.service';
import { VariableService } from '../../../../service/variable/variable.service';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { ParameterEvent } from '../../../parameter/parameter.event.model';
declare var CodeMirror: any;

@Component({
    selector: 'app-workflow-node-context',
    templateUrl: './node.context.html',
    styleUrls: ['./node.context.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeContextComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;
    @Input() loading: boolean;

    @Output() contextEvent = new EventEmitter<Workflow>();

    @ViewChild('nodeContextModal')
    public nodeContextModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    @ViewChild('textareaCodeMirror')
    codemirror: CodemirrorComponent;

    editableNode: WNode;

    suggest: string[] = [];
    payloadString: string;
    branches: string[] = [];
    remotes: string[] = [];
    tags: string[] = [];
    codeMirrorConfig: {};
    invalidJSON = false;
    loadingBranches = false;

    pipParamsReady = false;
    currentPipeline: Pipeline;
    pipelineSubscription: Subscription;
    permissionEnum = PermissionValue;

    constructor(
        private store: Store,
        private _variableService: VariableService,
        private _modalService: SuiModalService,
        private _appWorkflowService: ApplicationWorkflowService
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true
        };
    }

    show(): void {
        if (this.nodeContextModal) {
            this.suggest = [];
            this._variableService.getContextVariable(this.project.key, this.node.context.pipeline_id)
                .subscribe((suggest) => this.suggest = suggest);

            // TODO delete .repository_fullname condition and update handler to get history branches of node_run (issue: #1815)
            let app = Workflow.getApplication(this.workflow, this.node);
            if (this.node.context && app && app.repository_fullname) {
                this.loadingBranches = true;
                this.refreshVCSInfos(app);
            }

            this.editableNode = cloneDeep(this.node);
            if (!this.editableNode.context.default_payload) {
                this.editableNode.context.default_payload = {};
            }
            this.payloadString = JSON.stringify(this.editableNode.context.default_payload, undefined, 4);

            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeContextModal);
            this.modalConfig.mustScroll = true;
            this.modal = this._modalService.open(this.modalConfig);

            let pipeline = Workflow.getPipeline(this.workflow, this.node);
            if (pipeline) {
                this.store.dispatch(new FetchPipeline({
                    projectKey: this.project.key,
                    pipelineName: pipeline.name
                })).pipe(
                    flatMap(() => this.store.selectOnce(PipelinesState.selectPipeline(this.project.key, pipeline.name)))
                ).subscribe((pip) => {
                    this.currentPipeline = pip;
                    this.pipParamsReady = true;
                    this.editableNode.context.default_pipeline_parameters =
                        cloneDeep(Pipeline.mergeAndKeepOld(pip.parameters, this.editableNode.context.default_pipeline_parameters));
                    try {
                        this.editableNode.context.default_payload = JSON.parse(this.payloadString);
                        this.invalidJSON = false;
                    } catch (e) {
                        this.invalidJSON = true;
                    }
                    if (!this.editableNode.context.default_payload) {
                        this.editableNode.context.default_payload = {};
                    }
                });
            }
        }
    }

    refreshVCSInfos(app: Application, remote?: string) {
        this._appWorkflowService.getVCSInfos(this.project.key, app.name, remote)
            .pipe(finalize(() => this.loadingBranches = false))
            .subscribe((vcsInfos) => {
                if (vcsInfos.branches) {
                    this.branches = vcsInfos.branches.map((br) => '"' + br.display_id + '"');
                }
                if (vcsInfos.remotes) {
                    this.remotes = vcsInfos.remotes.map((rem) => '"' + rem.fullname + '"');
                }
                if (vcsInfos.tags) {
                    this.tags = vcsInfos.tags.map((tag) => '"' + tag.tag + '"');
                }
            });
    }

    saveContext(): void {
        if (this.editableNode.context.default_pipeline_parameters) {
            this.editableNode.context.default_pipeline_parameters.forEach(p => {
                p.value = p.value.toString();
            });
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let node = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        // If non root node
        if (this.editableNode.name !== clonedWorkflow.workflow_data.node.name) {
            this.editableNode.context.default_payload = null;
        }
        node.context = cloneDeep(this.editableNode.context);
        this.contextEvent.emit(clonedWorkflow);
    }

    reindent(): void {
        this.updateValue(this.payloadString);
    }

    updateValue(payload): void {
        let newPayload: {};
        if (!payload) {
            return;
        }

        try {
            newPayload = JSON.parse(payload);
            this.invalidJSON = false;
        } catch (e) {
            this.invalidJSON = true;
            return;
        }
        this.payloadString = JSON.stringify(newPayload, undefined, 4);
        this.editableNode.context.default_payload = JSON.parse(this.payloadString);
    }

    changeCodeMirror(eventRoot: Event): void {
        if (eventRoot.type !== 'click') {
            this.updateValue(eventRoot);
        }
        if (!this.codemirror || !this.codemirror.instance) {
            return;
        }

        if (eventRoot.type === 'click') {
            this.showHint(this.codemirror.instance, null);
        }
        this.codemirror.instance.on('keyup', (cm, event) => {
            if (!cm.state.completionActive && event.keyCode !== 32) {
                this.showHint(cm, event);
            }
        });
    }

    showHint(cm, event) {
        CodeMirror.showHint(this.codemirror.instance, CodeMirror.hint.payload, {
            completeSingle: true,
            closeCharacters: / /,
            payloadCompletionList: {
                branches: this.branches,
                repositories: this.remotes,
                tags: this.tags,
            },
            specialChars: ''
        });
    }

    parameterEvent(event: ParameterEvent) {
        switch (event.type) {
            case 'delete':
                this.editableNode.context.default_pipeline_parameters =
                    this.editableNode.context.default_pipeline_parameters.filter((param) => param.name !== event.parameter.name);
                event.parameter.updating = false;
                break;
        }
    }
}
