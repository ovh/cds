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
import { IdName, Project } from 'app/model/project.model';
import { WorkflowHookModel } from 'app/model/workflow.hook.model';
import { WNode, WNodeHook, WNodeOutgoingHook, WNodeType, Workflow } from 'app/model/workflow.model';
import { HookService } from 'app/service/hook/hook.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState } from 'app/store/project.state';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable, Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-outgoinghook',
    templateUrl: './wizard.outgoinghook.html',
    styleUrls: ['./wizard.outgoinghook.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWizardOutgoingHookComponent implements OnInit, OnDestroy {
    @ViewChild('textareaCodeMirror') codemirror: any;

    @Input() workflow: Workflow;
    @Input() mode = 'create'; // create / edit / ro
    @Input() display: boolean;

    @Output() outgoinghookEvent = new EventEmitter<WNode>();
    @Output() outgoinghookChange = new EventEmitter<boolean>();

    @Select(WorkflowState.getSelectedNode()) node$: Observable<WNode>;
    outgoingHook: WNode;
    nodeSub: Subscription;

    project: Project;
    editMode: boolean;

    codeMirrorConfig: any;
    loadingModels = false;
    outgoingHookModels: Array<WorkflowHookModel>;
    selectedOutgoingHookModel: WorkflowHookModel;
    displayConfig: boolean;
    availableWorkflows: Array<IdName>;
    loadingHooks: boolean;
    availableHooks: Array<WNodeHook>;
    invalidJSON = false;
    outgoing_default_payload: {};
    loading = false;
    codeMirrorSkipChange = true;
    themeSubscription: Subscription;

    constructor(
        private _store: Store,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _hookService: HookService,
        private _workflowService: WorkflowService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: this.mode === 'ro'
        };
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this.editMode = this._store.selectSnapshot(WorkflowState).editMode;
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        if (this.mode !== 'create') {
            this.nodeSub = this.node$.subscribe(n => {
                this.outgoingHook = cloneDeep(n);
                this._cd.markForCheck();
            });
        }

        this.themeSubscription = this._theme.get().pipe(finalize(() => this._cd.markForCheck())).subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });

        this.loadingModels = true;
        this._hookService.getOutgoingHookModel().pipe(finalize(() => {
            this.loadingModels = false;
            this._cd.markForCheck();
        }))
            .subscribe(ms => {
                this.outgoingHookModels = ms;
                if (this.outgoingHook) {
                    // select model
                    if (this.outgoingHook.outgoing_hook.hook_model_id) {
                        this.selectedOutgoingHookModel = this.outgoingHookModels
                            .find(m => m.id === this.outgoingHook.outgoing_hook.hook_model_id);
                    }
                    this.displayConfig = Object.keys(this.outgoingHook.outgoing_hook.config).length !== 0;
                    if (this.selectedOutgoingHookModel.name === 'Workflow') {
                        this.updateWorkflowData(false);
                    }
                    this.availableWorkflows = this.project.workflow_names.filter(idName => idName.name !== this.workflow.name);
                }
            });
    }

    updateOutgoingHook(): void {
        if (!this.outgoingHook) {
            this.outgoingHook = new WNode();
            this.outgoingHook.type = WNodeType.OUTGOINGHOOK;
            this.outgoingHook.outgoing_hook = new WNodeOutgoingHook();
        }
        this.outgoingHook.outgoing_hook.hook_model_id = this.selectedOutgoingHookModel.id;
        this.outgoingHook.outgoing_hook.config = cloneDeep(this.selectedOutgoingHookModel.default_config);
        this.outgoingHook.outgoing_hook.model = this.selectedOutgoingHookModel;
        this.displayConfig = Object.keys(this.outgoingHook.outgoing_hook.config).length !== 0;


        // Specific behavior for the 'workflow' hooks
        if (this.selectedOutgoingHookModel.name === 'Workflow') {
            // Current limitation: trigger only workflow in the same project
            this.outgoingHook.outgoing_hook.config['target_project'].value = this.project.key;
            // Load the workflow for the current project, but exclude the current workflow
            this.availableWorkflows = this.project.workflow_names.filter(idName => idName.name !== this.workflow.name);
        }
    }

    updateWorkflowData(pushChange: boolean): void {
        if (pushChange) {
            this.pushChange();
        }
        if (this.outgoingHook && this.outgoingHook.outgoing_hook.config && this.outgoingHook.outgoing_hook.config['target_project']
            && this.outgoingHook.outgoing_hook.config['target_workflow']) {
            this.loadingHooks = true;

            this._workflowService.getWorkflow(this.outgoingHook.outgoing_hook.config['target_project'].value,
                this.outgoingHook.outgoing_hook.config['target_workflow'].value)
                .pipe(first(), finalize(() => {
                    this.loadingHooks = false;
                    this._cd.markForCheck();
                }))
                .subscribe(wf => {
                    this.outgoing_default_payload = wf.workflow_data.node.context.default_payload;
                    let allHooks = Workflow.getAllHooks(wf);
                    if (allHooks) {
                        this.availableHooks = allHooks.filter(h => wf.hook_models[h.hook_model_id].name === 'Workflow');
                    } else {
                        this.availableHooks = [];
                    }
                    if (this.outgoingHook || this.outgoingHook.outgoing_hook.config['target_hook']) {
                        if (this.availableHooks
                            .findIndex(h =>
                                h.uuid === this.outgoingHook.outgoing_hook.config['target_hook'].value) === -1) {
                            this.outgoingHook.outgoing_hook.config['target_hook'].value = undefined;
                        }
                    }
                });
        } else {
            this.availableHooks = null;
        }
    }

    updateWorkflowOutgoingHook(): void {
        this.pushChange();
        if (this.outgoingHook.outgoing_hook.config['target_hook']) {
            this.outgoingHook.outgoing_hook.config['payload'].value = JSON.stringify(this.outgoing_default_payload, undefined, 4);
        }
    }

    updateWorkflow(): void {
        this.loading = true;
        let clonedWorkflow = cloneDeep(this.workflow);
        let n = Workflow.getNodeByID(this.outgoingHook.id, clonedWorkflow);

        n.outgoing_hook = this.outgoingHook.outgoing_hook;
        this._store.dispatch(new UpdateWorkflow({
            projectKey: this.workflow.project_key,
            workflowName: this.workflow.name,
            changes: clonedWorkflow
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(() => {
                this.outgoinghookChange.emit(false);
                if (this.editMode) {
                    this._toast.info('', this._translate.instant('workflow_ascode_updated'));
                } else {
                    this._toast.success('', this._translate.instant('workflow_updated'));
                }
            });
    }

    changeCodeMirror(code: string) {
        // skip first event that is initialization of codemirror
        if (!this.codeMirrorSkipChange) {
            this.pushChange();
        }
        if (this.codeMirrorSkipChange) {
            this.codeMirrorSkipChange = false;
        }


        if (typeof code === 'string') {
            this.invalidJSON = false;
            try {
                JSON.parse(code);
            } catch (e) {
                this.invalidJSON = true;
            }
        }
    }

    pushChange(): void {
        this.outgoinghookChange.emit(true);
    }
}
