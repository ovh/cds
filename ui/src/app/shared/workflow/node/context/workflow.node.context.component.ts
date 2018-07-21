import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {cloneDeep} from 'lodash';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Subscription} from 'rxjs';
import {finalize} from 'rxjs/operators';
import {Pipeline} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {Workflow, WorkflowNode} from '../../../../model/workflow.model';
import {ApplicationWorkflowService} from '../../../../service/application/application.workflow.service';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {VariableService} from '../../../../service/variable/variable.service';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {ParameterEvent} from '../../../parameter/parameter.event.model';
declare var CodeMirror: any;

@Component({
    selector: 'app-workflow-node-context',
    templateUrl: './node.context.html',
    styleUrls: ['./node.context.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeContextComponent {

    @Input() project: Project;
    @Input() node: WorkflowNode;
    @Input() workflow: Workflow;
    @Input() loading: boolean;

    @Output() contextEvent = new EventEmitter<WorkflowNode>();

    @ViewChild('nodeContextModal')
    public nodeContextModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    @ViewChild('textareaCodeMirror')
    codemirror: CodemirrorComponent;

    editableNode: WorkflowNode;

    suggest: string[] = [];
    payloadString: string;
    branches: string[] = [];
    codeMirrorConfig: {};
    invalidJSON = false;
    loadingBranches = false;

    pipParamsReady = false;
    currentPipeline: Pipeline;
    pipelineSubscription: Subscription;

    constructor(
      private _pipelineStore: PipelineStore,
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
            this._variableService.getContextVariable(this.project.key, this.node.pipeline_id)
              .subscribe((suggest) => this.suggest = suggest);

            // TODO delete .repository_fullname condition and update handler to get history branches of node_run (issue: #1815)
            if (this.node.context && this.node.context.application && this.node.context.application.repository_fullname) {
                this.loadingBranches = true;
                this._appWorkflowService.getBranches(this.project.key, this.node.context.application.name)
                    .pipe(finalize(() => this.loadingBranches = false))
                    .subscribe((branches) => this.branches = branches.map((br) => '"' + br.display_id + '"'));
            }

            this.editableNode = cloneDeep(this.node);
            if (!this.editableNode.context.default_payload) {
                this.editableNode.context.default_payload = {};
            }
            this.payloadString = JSON.stringify(this.editableNode.context.default_payload, undefined, 4);

            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeContextModal);
            this.modalConfig.mustScroll = true;
            this.modal = this._modalService.open(this.modalConfig);

            this.pipelineSubscription = this._pipelineStore.getPipelines(this.project.key, this.node.pipeline.name).subscribe(pips => {
                let pip = pips.get(this.project.key + '-' + this.node.pipeline.name);
                if (pip) {
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
                }
                if (this.pipelineSubscription) {
                  this.pipelineSubscription.unsubscribe();
                }
            });
        }
    }

    saveContext(): void {
        if (this.editableNode.context.project_platform_id === 0) {
            this.editableNode.context.project_platform = null;
        }
        if (this.editableNode.context.default_pipeline_parameters) {
            this.editableNode.context.default_pipeline_parameters.forEach(p => {
                p.value = p.value.toString();
            });
        }
        this.contextEvent.emit(this.editableNode);
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
            payloadCompletionList: this.branches,
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
