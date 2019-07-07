import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Parameter } from 'app/model/parameter.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Commit } from 'app/model/repositories.model';
import { WNode, WNodeContext, WNodeType, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowNodeRunManual, WorkflowRun, WorkflowRunRequest } from 'app/model/workflow.run.model';
import { ApplicationWorkflowService } from 'app/service/application/application.workflow.service';
import { ThemeStore } from 'app/service/services.module';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import cloneDeep from 'lodash-es/cloneDeep';
import { debounceTime, finalize, first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

declare var CodeMirror: any;

@Component({
    selector: 'app-workflow-node-run-param',
    templateUrl: './node.run.param.html',
    styleUrls: ['./node.run.param.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeRunParamComponent implements OnInit {
    @ViewChild('runWithParamModal', {static: false})
    runWithParamModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;

    @ViewChild('textareaCodeMirror', {static: false}) codemirror: any;

    @Input() workflowRun: WorkflowRun;
    _nodeRun: WorkflowNodeRun;
    @Input('nodeRun')
    set nodeRun(data: WorkflowNodeRun) {
        this._nodeRun = data;
    }
    get nodeRun() {
        return this._nodeRun;
    }
    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() num: number;

    @Input('nodeToRun')
    set nodeToRun(data: WNode) {
        if (data) {
            this._nodeToRun = cloneDeep(data);
            if (this.nodeRun && this.nodeRun.workflow_node_id !== this._nodeToRun.id) {
                delete this.nodeRun;
            }
            this.updateDefaultPipelineParameters();
            if (this._nodeToRun.context) {
                if ((!this.nodeRun || !this.nodeRun.payload) &&
                    (!this.workflowRun || !this.workflowRun.nodes ||
                        !this.workflowRun.nodes[this.workflowRun.workflow.workflow_data.node.id])) {
                    this.payloadString = JSON.stringify(this._nodeToRun.context.default_payload, undefined, 4);
                }
            }
        }
    }
    get nodeToRun(): WNode {
        return this._nodeToRun;
    }

    _nodeToRun: WNode;
    _previousBranch: string;
    _completionListener: any;
    _keyUpListener: any;
    _firstCommitLoad = false;

    lastNum: number;
    codeMirrorConfig: any;
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
    nodeTypeEnum = WNodeType;
    open: boolean;
    themeSubscription: Subscription;

    constructor(
        private _modalService: SuiModalService,
        private _workflowRunService: WorkflowRunService,
        private _router: Router,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _appWorkflowService: ApplicationWorkflowService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true
        };
    }

    ngOnInit(): void {
        this.linkedToRepo = WNode.linkedToRepo(this._nodeToRun, this.workflow);

        this.themeSubscription = this._theme.get()
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    show(): void {
        if (this.open) {
            return;
        }

        this.open = true;

        let num: number;
        let nodeRunID: number;

        if (this.nodeRun) { // relaunch a pipeline
            num = this.nodeRun.num;
            nodeRunID = this.nodeRun.id;
        } else if (this.workflowRun && this.workflowRun.nodes) {
            let rootNodeRun = this.workflowRun.nodes[this.workflowRun.workflow.workflow_data.node.id][0];
            num = rootNodeRun.num;
            if (this._nodeToRun.id === rootNodeRun.workflow_node_id) {
                nodeRunID = rootNodeRun.id;
            }
        }
        this.num = num;

        // if the pipeline was already launched, we refresh data from API

        // relaunch a workflow or a pipeline
        if (num > 0 && nodeRunID > 0) {
            this.readOnly = true;
            this._workflowRunService.getWorkflowNodeRun(
                this.project.key, this.workflow.name, num, nodeRunID)
                .pipe(finalize(() => this._cd.markForCheck()))
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
            } else if (this.workflowRun && this.workflowRun.workflow.workflow_data.node.id === this.nodeToRun.id) {
                isPipelineRoot = true;
            }
            // run a workflow or a child pipeline, first run
            let payload = null;
            this.readOnly = false;
            // if it's not the pipeline root, we take the payload on the pipelineRoot
            if (!isPipelineRoot) {
                this.readOnly = true;
                let rootNodeRun = this.workflowRun.nodes[this.workflowRun.workflow.workflow_data.node.id][0];
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
            currentPayload = cloneDeep(this.getCurrentPayload());
            if (this.readOnly) {
                delete currentPayload['payload'];
            }
            this.payloadString = JSON.stringify(currentPayload, undefined, 4);
        }

        this.modal = this._modalService.open(config);
        this.modal.onApprove(() => { this.open = false; });
        this.modal.onDeny(() => { this.open = false; });

        this.codeMirrorConfig = Object.assign({}, this.codeMirrorConfig, { readOnly: this.readOnly });

        if (!this.nodeToRun.context || !this.nodeToRun.context.application_id) {
            return;
        }

        if (this.linkedToRepo) {
            if (this.workflow.applications[this.nodeToRun.context.application_id].repository_fullname) {
                this.loadingBranches = true;
                this.refreshVCSInfos();
            }

            if (this.num == null) {
                this.loadingCommits = true;
                this._workflowRunService.getRunNumber(this.project.key, this.workflow)
                    .pipe(first(), finalize(() => this._cd.markForCheck()))
                    .subscribe(n => {
                        this.lastNum = n.num + 1;
                        this.getCommits(n.num + 1, false);
                    });
            } else {
                this.getCommits(this.num, false);
            }
        }
    }

    getCommits(num: number, change: boolean) {
        if (!this.linkedToRepo) {
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

        if (num == null) {
            return;
        }

        this._firstCommitLoad = true;
        this._previousBranch = branch;
        this.loadingCommits = true;
        this._workflowRunService.getCommits(this.project.key, this.workflow.name, num, this.nodeToRun.name, branch, hash, repository)
            .pipe(
                debounceTime(500),
                finalize(() => {
                    this.loadingCommits = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe((commits) => this.commits = commits);
    }

    getCurrentPayload(): {} {
        let currentContext = {};
        if (this.nodeRun && this.nodeRun.payload) {
            currentContext = this.nodeRun.payload;
        } else if (this.workflowRun && this.workflowRun.nodes) {
            let rootNodeRun = this.workflowRun.nodes[this.workflowRun.workflow.workflow_data.node.id];
            if (rootNodeRun) {
                currentContext = rootNodeRun[0].payload;
            }
        } else if (this.nodeToRun.context) {
            if (this.nodeToRun.context.default_payload && Object.keys(this.nodeToRun.context.default_payload).length > 0) {
                currentContext = this.nodeToRun.context.default_payload;
            } else {
                currentContext = this.workflow.workflow_data.node.context.default_payload;
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
            this._nodeToRun.context = new WNodeContext();
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
                this._cd.markForCheck();
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

        // TODO SIMPLIFY AFTER MIGRATION
        if (this.nodeRun) {
            request.number = this.nodeRun.num;
            request.from_nodes = [this.nodeRun.workflow_node_id];
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
                { queryParams: { subnum: wr.last_subnumber } });
            wr.force_update = true;
        });
    }

    refreshVCSInfos(remote?: string) {
        this._appWorkflowService.getVCSInfos(this.project.key,
            this.workflow.applications[this.nodeToRun.context.application_id].name, remote)
            .pipe(finalize(() => {
                this.loadingBranches = false;
                this._cd.markForCheck();
            }))
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
                if (!this.linkedToRepo) {
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
