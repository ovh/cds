import {Component, Input, ViewChild} from '@angular/core';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Workflow, WorkflowNode, WorkflowNodeContext} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';
import {cloneDeep} from 'lodash';
import {Pipeline} from '../../../../model/pipeline.model';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {WorkflowNodeRunManual, WorkflowRunRequest} from '../../../../model/workflow.run.model';
import {Router} from '@angular/router';

@Component({
    selector: 'app-workflow-node-run-param',
    templateUrl: './node.run.param.html',
    styleUrls: ['./node.run.param.scss']
})
export class WorkflowNodeRunParamComponent {

    @ViewChild('runWithParamModal')
    runWithParamModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input('nodeToRun')
    set nodeToRun (data: WorkflowNode) {
        this._nodeToRun = cloneDeep(data);
        this.updateDefaultPipelineParameters();
        if (this._nodeToRun.context) {
            this.payloadString = JSON.stringify(this._nodeToRun.context.default_payload);
        }
    };
    get nodeToRun(): WorkflowNode {
        return this._nodeToRun;
    }
    _nodeToRun: WorkflowNode;

    codeMirrorConfig: {};
    payloadString: string;
    invalidJSON: boolean;

    constructor(private _modalService: SuiModalService, private _workflowRunService: WorkflowRunService, private _router: Router) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true
        };
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

    run(): void {
        let request = new WorkflowRunRequest();
        request.manual = new WorkflowNodeRunManual();
        request.manual.payload = this.payloadString;
        request.manual.pipeline_parameter = this.nodeToRun.context.default_pipeline_parameters;
        this._workflowRunService.runWorkflow(this.project.key, this.workflow, request).subscribe(wr => {
            this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name, 'run', wr.num]);
        });
    }
}
