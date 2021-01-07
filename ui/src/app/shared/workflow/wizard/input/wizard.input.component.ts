import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnDestroy,
    OnInit,
    Output,
    ViewChild
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun } from 'app/model/workflow.run.model';
import { ApplicationWorkflowService } from 'app/service/application/application.workflow.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { VariableService } from 'app/service/variable/variable.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ParameterEvent } from 'app/shared/parameter/parameter.event.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState } from 'app/store/project.state';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

declare let CodeMirror: any;

@Component({
    selector: 'app-workflow-node-input',
    templateUrl: './wizard.input.html',
    styleUrls: ['./wizard.input.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWizardNodeInputComponent implements OnInit, OnDestroy {

    @Input() workflow: Workflow;
    @Input() readonly = true;
    @Output() inputChange = new EventEmitter<boolean>();

    @ViewChild('textareaCodeMirror') codemirror: any;

    project: Project;
    editMode: boolean;
    noderun: WorkflowNodeRun;

    @Select(WorkflowState.getSelectedNode()) node$: Observable<WNode>;
    editableNode: WNode;
    nodeSub: Subscription;

    suggest: string[] = [];
    payloadString: string;
    branches: string[] = [];
    invalidJSON = false;
    loadingBranches = false;
    codeMirrorConfig: any;
    pipParamsReady = false;
    currentPipeline: Pipeline;
    remotes: string[] = [];
    tags: string[] = [];
    loading: boolean;
    themeSubscription: Subscription;

    constructor(
        private store: Store,
        private _variableService: VariableService,
        private _appWorkflowService: ApplicationWorkflowService,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef,
        private _pipelineService: PipelineService
    ) {
        this.project = this.store.selectSnapshot(ProjectState.projectSnapshot);
        this.editMode = this.store.selectSnapshot(WorkflowState).editMode;
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.noderun = this.store.selectSnapshot(WorkflowState).workflowNodeRun;
        this.nodeSub = this.node$.subscribe(n => {
            this.editableNode = cloneDeep(n);
            if (this.editableNode) {
                this.init();
            } else {
                this.payloadString = JSON.stringify(this.noderun.payload, undefined, 4);
            }
            this._cd.markForCheck();
        });



        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: this.readonly
        };

        this.themeSubscription = this._theme.get().pipe(finalize(() => this._cd.markForCheck())).subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    init(): void {
        if (!this.editableNode.context.default_payload) {
            this.editableNode.context.default_payload = {};
        }

        this.suggest = [];
        this._variableService.getContextVariable(this.project.key, this.editableNode.context.pipeline_id)
            .subscribe((suggest) => this.suggest = suggest);

        // TODO delete .repository_fullname condition and update handler to get history branches of node_run (issue: #1815)
        let app = Workflow.getApplication(this.workflow, this.editableNode);
        if (this.editableNode.context && app && app.repository_fullname) {
            this.loadingBranches = true;
            this.refreshVCSInfos(app);
        }

        this.payloadString = JSON.stringify(this.editableNode.context.default_payload, undefined, 4);
        let pipeline = Workflow.getPipeline(this.workflow, this.editableNode);
        if (pipeline) {
            this._pipelineService.getPipeline(this.project.key, pipeline.name)
                .pipe(first()).subscribe(p => {
                    this.currentPipeline = p;
                    this.pipParamsReady = true;
                    this.editableNode.context.default_pipeline_parameters =
                        cloneDeep(Pipeline.mergeAndKeepOld(p.parameters, this.editableNode.context.default_pipeline_parameters));
                    try {
                        this.editableNode.context.default_payload = JSON.parse(this.payloadString);
                        this.invalidJSON = false;
                    } catch (e) {
                        this.invalidJSON = true;
                    }
                    if (!this.editableNode.context.default_payload) {
                        this.editableNode.context.default_payload = {};
                    }
                    this._cd.markForCheck();
            });
        }
    }

    reindent(): void {
        this.updateValue(this.payloadString);
    }

    pushEvent(): void {
        this.inputChange.emit(true);
    }

    parameterEvent(event: ParameterEvent) {
        this.pushEvent();
    }

    changeCodeMirror(eventRoot: Event, sendEvent: boolean): void {
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

    refreshVCSInfos(app: Application, remote?: string) {
        this._appWorkflowService.getVCSInfos(this.project.key, app.name, remote)
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

    updateValue(payload): void {
        if (this.noderun) {
            return;
        }
        let newPayload: {};
        if (!payload) {
            return;
        }

        let previousPayload = JSON.stringify(cloneDeep(this.editableNode.context.default_payload), undefined, 4);
        try {
            newPayload = JSON.parse(payload);
            this.invalidJSON = false;
        } catch (e) {
            this.invalidJSON = true;
            return;
        }
        this.payloadString = JSON.stringify(newPayload, undefined, 4);
        this.editableNode.context.default_payload = JSON.parse(this.payloadString);

        if (this.payloadString !== previousPayload) {
            this.pushEvent();
        }
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

    updateWorkflow(): void {
        this.loading = true;
        let clonedWorkflow = cloneDeep(this.workflow);
        let n: WNode;
        if (this.editMode) {
            n = Workflow.getNodeByRef(this.editableNode.ref, clonedWorkflow);
        } else {
            n = Workflow.getNodeByID(this.editableNode.id, clonedWorkflow);
        }

        n.context.default_payload = this.editableNode.context.default_payload;
        n.context.default_pipeline_parameters = this.editableNode.context.default_pipeline_parameters;
        if (n.context.default_pipeline_parameters) {
            n.context.default_pipeline_parameters.forEach(p => {
               p.value = p.value.toString();
            });
        }
        this.store.dispatch(new UpdateWorkflow({
            projectKey: this.workflow.project_key,
            workflowName: this.workflow.name,
            changes: clonedWorkflow
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this.inputChange.emit(false);
                if (this.editMode) {
                    this._toast.info('', this._translate.instant('workflow_ascode_updated'));
                } else {
                    this._toast.success('', this._translate.instant('workflow_updated'));
                }
            });
    }
}
