import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {Workflow, WorkflowNode} from '../../../../model/workflow.model';
import {Pipeline} from '../../../../model/pipeline.model';
import {cloneDeep} from 'lodash';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {Subscription} from 'rxjs/Subscription';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';

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

    editableNode: WorkflowNode;

    payloadString: string;
    codeMirrorConfig: {};
    invalidJSON = false;

    pipParamsReady = false;
    pipelineSubscription: Subscription;

    constructor(private _pipelineStore: PipelineStore, private _modalService: SuiModalService) {
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
            this.editableNode = cloneDeep(this.node);
            if (!this.editableNode.context.default_payload) {
                this.editableNode.context.default_payload = {};
            }
            this.payloadString = JSON.stringify(this.editableNode.context.default_payload);

            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeContextModal);
            this.modal = this._modalService.open(this.modalConfig);

            this.pipelineSubscription = this._pipelineStore.getPipelines(this.project.key, this.node.pipeline.name).subscribe(pips => {
                let pip = pips.get(this.project.key + '-' + this.node.pipeline.name);
                if (pip) {
                    this.pipParamsReady = true;
                    this.editableNode.context.default_pipeline_parameters =
                        Pipeline.mergeParams(pip.parameters, this.editableNode.context.default_pipeline_parameters);
                    this.editableNode.context.default_payload = JSON.stringify(this.editableNode.context.default_payload, undefined, 4);
                    if (!this.editableNode.context.default_payload) {
                        this.editableNode.context.default_payload = '{}';
                    }
                }
                this.pipelineSubscription.unsubscribe();
            });
        }
    }

    saveContext(): void {
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
}
