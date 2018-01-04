import {Component, Input, ViewChild} from '@angular/core';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Workflow, WorkflowNode, WorkflowNodeContext} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';
import {cloneDeep} from 'lodash';
import {Pipeline} from '../../../../model/pipeline.model';
import {Commit} from '../../../../model/repositories.model';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {ApplicationWorkflowService} from '../../../../service/application/application.workflow.service';
import {WorkflowNodeRun, WorkflowNodeRunManual, WorkflowRunRequest} from '../../../../model/workflow.run.model';
import {Router} from '@angular/router';
import {WorkflowCoreService} from '../../../../service/workflow/workflow.core.service';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
import {finalize, first, debounceTime} from 'rxjs/operators';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../toast/ToastService';
declare var CodeMirror: any;

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

    @ViewChild('textareaCodeMirror')
    codemirror: CodemirrorComponent;

    @Input() canResync = false;
    @Input() nodeRun: WorkflowNodeRun;
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() num: number;

    @Input('nodeToRun')
    set nodeToRun(data: WorkflowNode) {
        if (data) {
            this._nodeToRun = cloneDeep(data);
            this.updateDefaultPipelineParameters();
            if (this._nodeToRun.context) {
                this.payloadString = JSON.stringify(this._nodeToRun.context.default_payload);
                if (this.payloadString) {
                    this.reindent();
                }
            }
        }
    }
    get nodeToRun(): WorkflowNode {
        return this._nodeToRun;
    }

    _nodeToRun: WorkflowNode;

    lastNum: number;
    codeMirrorConfig: {};
    commits: Commit[] = [];
    branches: string[] = [];
    payloadString: string;
    invalidJSON: boolean;
    isSync = false;
    loading = false;
    loadingCommits = false;
    loadingBranches = false;

    constructor(private _modalService: SuiModalService, private _workflowRunService: WorkflowRunService, private _router: Router,
                private _workflowCoreService: WorkflowCoreService, private _translate: TranslateService, private _toast: ToastService,
                private _appWorkflowService: ApplicationWorkflowService) {
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

        if (!this.nodeToRun.context.application) {
            return;
        }

        // TODO delete .repository_fullname condition and update handler to get history branches of node_run (issue: #1815)
        if (this.nodeToRun.context.application.repository_fullname) {
            this.loadingBranches = true;
            this._appWorkflowService.getBranches(this.project.key, this.nodeToRun.context.application.name)
                .pipe(finalize(() => this.loadingBranches = false))
                .subscribe((branches) => this.branches = branches.map((br) => '"' + br.display_id + '"'));
        }

        if (this.num == null) {
            this.loadingCommits = true;
            this._workflowRunService.getRunNumber(this.project.key, this.workflow)
                .pipe(
                    first()
                )
                .subscribe(n => {
                    this.lastNum = n.num + 1;
                    this.getCommits(n.num + 1);
                });
            return;
        }
        this.getCommits(this.num);
    }

    getCommits(num: number) {
        let branch;
        let currentContext;
        if (this.nodeToRun.context && this.nodeToRun.context.default_payload) {
            currentContext = this.nodeToRun.context.default_payload;
        }

        try {
            currentContext = JSON.parse(this.payloadString);
            this.invalidJSON = false;
        } catch (e) {
            this.invalidJSON = true;
        }

        branch = currentContext['git.branch'];
        if (branch && !this.loadingBranches && this.branches.indexOf('"' + branch + '"') === -1) {
            return;
        }
        this.loadingCommits = true;
        this._workflowRunService.getCommits(this.project.key, this.workflow.name, num, this.nodeToRun.name, branch)
          .pipe(
            debounceTime(500),
            finalize(() => this.loadingCommits = false)
          )
          .subscribe((commits) => this.commits = commits);
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
        let num = this.num;
        if (this.nodeRun) {
            num = this.nodeRun.num;
        }
        this.loading = true;
        this._workflowRunService.resync(this.project.key, this.workflow, num)
        .pipe(finalize(() => {
            this.loading = false;
        })).subscribe(wr => {
            this.nodeToRun = Workflow.getNodeByID(this._nodeToRun.id, wr.workflow);
            this._toast.success('', this._translate.instant('workflow_run_resync_done'));
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
        } else if (this.nodeToRun && this.num != null) {
            request.from_nodes = [this.nodeToRun.id];
            request.number = this.num;
        }

        this.loading = true;
        this._workflowRunService.runWorkflow(this.project.key, this.workflow.name, request).pipe(finalize(() => {
            this.loading = false;
        })).subscribe(wr => {
            this.modal.approve(true);
            this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name, 'run', wr.num],
                {queryParams: {subnum: wr.last_subnumber}});
            this._workflowCoreService.setCurrentWorkflowRun(wr);
        });
    }

    changeCodeMirror(eventRoot: Event): void {
        let num = this.num;
        if (!this.codemirror || !this.codemirror.instance) {
            return
        }
        if (eventRoot.type === 'click') {
            this.showHint(this.codemirror.instance, null);
        }
        this.codemirror.instance.on('keyup', (cm, event) => {
            if (!cm.state.completionActive && event.keyCode !== 32) {
                this.showHint(cm, event);
            }
        });

        this.codemirror.instance.on('endCompletion', (cm, event) => {
            this.getCommits(num || this.lastNum);
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
}
