import {AfterViewInit, Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {Workflow, WorkflowNode} from '../../../../model/workflow.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Pipeline} from '../../../../model/pipeline.model';
import {cloneDeep} from 'lodash';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {Subscription} from 'rxjs/Subscription';

declare var CodeMirror: any;

@Component({
    selector: 'app-workflow-node-context',
    templateUrl: './node.context.html',
    styleUrls: ['./node.context.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeContextComponent implements OnInit, AfterViewInit {

    @Input() project: Project;
    @Input() node: WorkflowNode;
    @Input() workflow: Workflow;

    @Output() contextEvent = new EventEmitter<WorkflowNode>();

    @ViewChild('nodeContextModal')
    nodeContextModal: SemanticModalComponent;

    editableNode: WorkflowNode;

    codeMirrorConfig: {};
    invalidJSON = false;

    pipParamsReady = false;
    pipelineSubscription: Subscription;

    constructor(private _pipelineStore: PipelineStore) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true
        };
    }

    ngOnInit(): void {
        this.editableNode = cloneDeep(this.node);
    }

    ngAfterViewInit(): void {
        this.nodeContextModal.onModalShow.subscribe(() => {
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
            });
        });
    }



    show(data?: {}): void {
        if (this.nodeContextModal) {
            this.nodeContextModal.show(data);
        }
    }

    hide(): void {
        if (this.nodeContextModal) {
            this.nodeContextModal.hide();
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
        this.editableNode.context.default_payload = JSON.stringify(this.editableNode.context.default_payload, undefined, 4);
    }

    updateValue(): void {
        try {
            JSON.parse(this.editableNode.context.default_payload);
            this.invalidJSON = false;
        } catch (e) {
            this.invalidJSON = true;
            return;
        }
    }
}
