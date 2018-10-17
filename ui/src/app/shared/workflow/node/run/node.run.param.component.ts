import {Component, Input, ViewChild} from '@angular/core';
import {Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {CodemirrorComponent} from 'ng2-codemirror-typescript/Codemirror';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {debounceTime, finalize, first} from 'rxjs/operators';
import {Parameter} from '../../../../model/parameter.model';
import {Pipeline} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {Commit} from '../../../../model/repositories.model';
import {Workflow, WorkflowNode, WorkflowNodeContext} from '../../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowNodeRunManual, WorkflowRun, WorkflowRunRequest} from '../../../../model/workflow.run.model';
import {ApplicationWorkflowService} from '../../../../service/application/application.workflow.service';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {WorkflowEventStore} from '../../../../service/workflow/workflow.event.store';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';
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

    @Input() workflowRun: WorkflowRun;
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
                if ((!this.nodeRun || !this.nodeRun.payload ) &&
                    (!this.workflowRun  || !this.workflowRun.nodes || !this.workflowRun.nodes[this.workflowRun.workflow.root.id])) {
                        this.payloadString = JSON.stringify(this._nodeToRun.context.default_payload, undefined, 4);
                }
                this.linkedToRepo = this._nodeToRun.context.application != null
                    && this._nodeToRun.context.application.repository_fullname != null;
            }
        }
    }
    get nodeToRun(): WorkflowNode {
        return this._nodeToRun;
    }

    _nodeToRun: WorkflowNode;
    _previousBranch: string;
    _completionListener: any;
    _keyUpListener: any;
    _firstCommitLoad = false;

    lastNum: number;
    codeMirrorConfig: {};
    commits: Commit[] = [];
    parameters: Parameter[] = [];
    branches: string[] = [];
    remotes: string[] = [];
    tags: string[] = [];
    payloadRemote: string;
    payloadString: string;
    invalidJSON: boolean;
    isSync = false;
    loading = false;
    loadingCommits = false;
    loadingBranches = false;
    readOnly = false;
    linkedToRepo = false;

    constructor(private _modalService: SuiModalService, private _workflowRunService: WorkflowRunService, private _router: Router,
                private _workflowEventStore: WorkflowEventStore, private _translate: TranslateService, private _toast: ToastService,
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
        let num: number;
        let nodeRunID: number;

        if (this.nodeRun) { // relaunch a pipeline
            num = this.nodeRun.num;
            nodeRunID = this.nodeRun.id;
        } else if (this.workflowRun && this.workflowRun.nodes) {
            let rootNodeRun = this.workflowRun.nodes[this.workflowRun.workflow.root.id][0];
            num = rootNodeRun.num;
            nodeRunID = rootNodeRun.id;
        }

        // if the pipeline was already launched, we refresh data from API

        // relaunch a workflow or a pipeline
        if (num > 0 && nodeRunID > 0) {
            this.readOnly = true;
            this._workflowRunService.getWorkflowNodeRun(
                this.project.key, this.workflow.name, num, nodeRunID)
                .subscribe(nodeRun => {
                    if (nodeRun && nodeRun.hook_event) {
                        this._nodeToRun.context.default_payload = nodeRun.hook_event.payload;
                        if (this.nodeRun) {
                            this._nodeToRun.context.default_pipeline_parameters = nodeRun.hook_event.pipeline_parameter;
                        }
                    }
                    if (nodeRun && nodeRun.manual) {
                        this._nodeToRun.context.default_payload = nodeRun.manual.payload;
                        if (this.nodeRun) {
                            this._nodeToRun.context.default_pipeline_parameters = nodeRun.manual.pipeline_parameter;
                        }
                    }
                    this.prepareDisplay(null);
            });
        } else {
            let isPipelineRoot = false;
            if (!this.workflowRun) {
                isPipelineRoot = true;
            } else if (this.workflowRun && this.workflowRun.workflow.root_id === this.nodeToRun.id) {
                isPipelineRoot = true;
            }
            // run a workflow or a child pipeline, first run
            let payload = null;
            this.readOnly = false;
            // if it's not the pipeline root, we take the payload on the pipelineRoot
            if (!isPipelineRoot) {
                this.readOnly = true;
                let rootNodeRun = this.workflowRun.nodes[this.workflowRun.workflow.root.id][0];
                payload = rootNodeRun.payload;
            }
            this.prepareDisplay(payload);
        }
    }

    prepareDisplay(payload): void {
        this._firstCommitLoad = false;
        this._previousBranch = null;
        const config = new TemplateModalConfig<boolean, boolean, void>(this.runWithParamModal);
        config.mustScroll = true;

        let currentPayload = payload;
        if (!currentPayload) {
            currentPayload = this.getCurrentPayload();
        }

        this.payloadString = JSON.stringify(currentPayload, undefined, 4);

        this.modal = this._modalService.open(config);

        this.codeMirrorConfig = Object.assign({}, this.codeMirrorConfig, {readOnly: this.readOnly});

        if (!this.nodeToRun.context.application) {
            return;
        }

        if (this.linkedToRepo) {
            if (this.nodeToRun.context.application.repository_fullname) {
                this.loadingBranches = true;
                this.refreshVCSInfos();
            }

            if (this.num == null) {
                this.loadingCommits = true;
                this._workflowRunService.getRunNumber(this.project.key, this.workflow)
                    .pipe(first())
                    .subscribe(n => {
                        this.lastNum = n.num + 1;
                        this.getCommits(n.num + 1, false);
                    });
            }

            if (this.num != null) {
                this.getCommits(this.num, false);
            }
        }
    }

    getCommits(num: number, change: boolean) {
        if (!WorkflowNode.isLinkedToRepo(this.nodeToRun)) {
            return;
        }
        let branch, hash, repository;
        let currentContext = this.getCurrentPayload();

        if (change && this.payloadString) {
            try {
                currentContext = JSON.parse(this.payloadString);
                this.invalidJSON = false;
            } catch (e) {
                this.invalidJSON = true;
                return;
            }
        }

        if (currentContext) {
          repository = currentContext['git.repository'];
          branch = currentContext['git.branch'];
          hash = currentContext['git.hash'];
        }

        if (this._firstCommitLoad && branch === this._previousBranch) {
            return;
        }

        if (this._firstCommitLoad && branch && !this.loadingBranches && this.branches.indexOf('"' + branch + '"') === -1) {
            return;
        }

        this._firstCommitLoad = true;
        this._previousBranch = branch;
        this.loadingCommits = true;
        this._workflowRunService.getCommits(this.project.key, this.workflow.name, num, this.nodeToRun.name, branch, hash, repository)
          .pipe(
            debounceTime(500),
            finalize(() => this.loadingCommits = false)
          )
          .subscribe((commits) => this.commits = commits);
    }

    getCurrentPayload(): {} {
        let currentContext = {};
        if (this.nodeRun && this.nodeRun.payload) {
            currentContext = this.nodeRun.payload;
        } else if (this.workflowRun && this.workflowRun.nodes) {
            let rootNodeRun = this.workflowRun.nodes[this.workflowRun.workflow.root.id];
            if (rootNodeRun) {
                currentContext = rootNodeRun[0].payload;
            }
        } else if (this.nodeToRun.context) {
            if (this.nodeToRun.context.default_payload && Object.keys(this.nodeToRun.context.default_payload).length > 0) {
                currentContext = this.nodeToRun.context.default_payload;
            } else {
                currentContext = this.workflow.root.context.default_payload;
            }
        }

        return currentContext;
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
    }

    private updateDefaultPipelineParameters() {
        let pipToRun = Workflow.getPipeline(this.workflow, this._nodeToRun);
        if (!pipToRun) {
          return;
        }

        if (this._nodeToRun.context) {
            this.parameters =
                Pipeline.mergeParams(cloneDeep(pipToRun.parameters), this._nodeToRun.context.default_pipeline_parameters);
        } else {
            this._nodeToRun.context = new WorkflowNodeContext();
            this.parameters = cloneDeep(pipToRun.parameters);
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
        request.manual.pipeline_parameter = Parameter.formatForAPI(this.parameters);

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
            wr.force_update = true;
            this._workflowEventStore.setSelectedRun(wr);
        });
    }

    refreshVCSInfos(remote?: string) {
        this._appWorkflowService.getVCSInfos(this.project.key, this.nodeToRun.context.application.name, remote)
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

    changeCodeMirror(eventRoot: Event): void {
        let num = this.num;
        if (!this.codemirror || !this.codemirror.instance) {
            return
        }
        if (eventRoot.type === 'click') {
            this.showHint(this.codemirror.instance, null);
        }

        if (!this._keyUpListener) {
            this._keyUpListener = this.codemirror.instance.on('keyup', (cm, event) => {
                if (!cm.state.completionActive && event.keyCode !== 32) {
                    this.showHint(cm, event);
                }
            });
        }

        if (!this._completionListener) {
            this._completionListener = this.codemirror.instance.on('endCompletion', (cm, event) => {
                if (!WorkflowNode.isLinkedToRepo(this.nodeToRun)) {
                    return;
                }
                let currentContext = this.getCurrentPayload();
                if (this.payloadString) {
                    try {
                        currentContext = JSON.parse(this.payloadString);
                        this.invalidJSON = false;
                    } catch (e) {
                        this.invalidJSON = true;
                        return;
                    }
                }
                let change = false;
                if (currentContext) {
                    change = currentContext['git.repository'] !== this.payloadRemote;
                    this.payloadRemote = currentContext['git.repository'];
                }
                if (change) {
                    this.refreshVCSInfos(this.payloadRemote);
                }

                this.getCommits(num || this.lastNum, true);
            });
        }
    }

    showHint(cm, event) {
        CodeMirror.showHint(this.codemirror.instance, CodeMirror.hint.payload, {
            completeSingle: true,
            closeCharacters: / /,
            payloadCompletionList: {
              branches: this.branches,
              tags: this.tags,
              repositories: this.remotes,
            },
            specialChars: ''
        });
    }
}
