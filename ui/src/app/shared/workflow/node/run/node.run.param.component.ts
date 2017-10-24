import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Workflow, WorkflowNode, WorkflowNodeContext} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';
import {cloneDeep} from 'lodash';
import {Pipeline} from '../../../../model/pipeline.model';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {WorkflowNodeRun, WorkflowNodeRunManual, WorkflowRunRequest} from '../../../../model/workflow.run.model';
import {Router} from '@angular/router';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-node-run-param',
    templateUrl: './node.run.param.html',
    styleUrls: ['./node.run.param.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeRunParamComponent {

    @ViewChild('runWithParamModal')
    runWithParamModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Input() canResync = false;
    @Input() nodeRun: WorkflowNodeRun;
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() num: number;
    @Input('nodeToRun')
    set nodeToRun (data: WorkflowNode) {
        if (data) {
            this._nodeToRun = cloneDeep(data);
            this.updateDefaultPipelineParameters();
            if (this._nodeToRun.context) {
                this.payloadString = JSON.stringify(this._nodeToRun.context.default_payload);
            }
            this.getPipeline();
        }
    };
    get nodeToRun(): WorkflowNode {
        return this._nodeToRun;
    }
    _nodeToRun: WorkflowNode;

    codeMirrorConfig: {};
    payloadString: string;
    invalidJSON: boolean;
    isSync = false;
    loading = false;

    pipelineSubscription: Subscription;

    constructor(private _modalService: SuiModalService, private _workflowRunService: WorkflowRunService, private _router: Router,
                private _pipStore: PipelineStore) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true
        };
    }

    getPipeline(): void {
        if (!this.pipelineSubscription) {
            this.pipelineSubscription = this._pipStore.getPipelines(this.project.key, this.nodeToRun.pipeline.name).subscribe(ps => {
                let pipkey = this.project.key + '-' + this.nodeToRun.pipeline.name;
                let pip = ps.get(pipkey);
                if (pip) {
                    if (pip.last_modified === this.nodeToRun.pipeline.last_modified) {
                        this.isSync = true;
                    }
                }
            });
        }

    }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.runWithParamModal);
        this.modal = this._modalService.open(config);
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
    }

    private updateDefaultPipelineParameters() {
        if (this._nodeToRun.context) {
            this._nodeToRun.context.default_pipeline_parameters =
                Pipeline.mergeParams(this._nodeToRun.pipeline.parameters, this._nodeToRun.context.default_pipeline_parameters);
        } else {
            this._nodeToRun.context = new WorkflowNodeContext();
            this._nodeToRun.context.default_pipeline_parameters = this._nodeToRun.pipeline.parameters;
        }

    }

    resync(): void {
        this._workflowRunService.resync(this.project.key, this.workflow, this.nodeRun.num).subscribe(wr => {
            this.nodeToRun = Workflow.getNodeByID(this._nodeToRun.id, wr.workflow);
            this.isSync = true;
        });
    }

    run(): void {
        if (this.payloadString && this.payloadString !== '') {
            this.reindent();
            if (this.invalidJSON) {
                return;
            }
        }
        let request = new WorkflowRunRequest();
        request.manual = new WorkflowNodeRunManual();
        request.manual.payload = this.payloadString ? JSON.parse(this.payloadString) : null;
        request.manual.pipeline_parameter = this.nodeToRun.context.default_pipeline_parameters;

        if (this.nodeRun) {
            request.from_nodes = [this.nodeRun.workflow_node_id];
            request.number = this.nodeRun.num;
        } else if (this.nodeToRun && this.num) {
            request.from_nodes = [this.nodeToRun.id];
            request.number = this.num;
        }

        this.loading = true;
        this._workflowRunService.runWorkflow(this.project.key, this.workflow.name, request).finally(() => {
            this.loading = false;
        }).subscribe(wr => {
            this.modal.approve(true);
            this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name, 'run', wr.num],
            {queryParams: { subnum: wr.last_subnumber }});
        });
    }
}
