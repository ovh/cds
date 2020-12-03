import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnDestroy, OnInit, Output, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
// eslint-disable-next-line max-len
import { WNode, WNodeContext, WNodeHook, Workflow, WorkflowNodeCondition, WorkflowNodeConditions, WorkflowTriggerConditionCache } from 'app/model/workflow.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Table } from 'app/shared/table/table';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState } from 'app/store/project.state';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import uniqBy from 'lodash-es/uniqBy';
import { Observable } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-node-conditions',
    templateUrl: './wizard.conditions.html',
    styleUrls: ['./wizard.conditions.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWizardNodeConditionComponent extends Table<WorkflowNodeCondition> implements OnInit, OnDestroy {
    @Input() workflow: Workflow;
    @Input() readonly = true;
    @Output() conditionsChange = new EventEmitter<boolean>();


    @ViewChild('textareaCodeMirror') codemirror: any;

    @Select(WorkflowState.getSelectedNode()) node$: Observable<WNode>;
    editableNode: WNode;
    nodeSub: Subscription;

    @Select(WorkflowState.getSelectedHook()) hook$: Observable<WNodeHook>;
    editableHook: WNodeHook;
    hookSub: Subscription;


    project: Project;
    editMode: boolean;

    codeMirrorConfig: any;
    loadingConditions = false;
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];
    loading = false;
    previousValue: string;
    themeSubscription: Subscription;
    triggerConditions: WorkflowTriggerConditionCache;

    constructor(
        private store: Store,
        private _workflowService: WorkflowService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        super();
        this.project = this.store.selectSnapshot(ProjectState.projectSnapshot);
        this.editMode = this.store.selectSnapshot(WorkflowState).editMode;
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    getData(): Array<WorkflowNodeCondition> {
        return undefined;
    }

    ngOnInit(): void {
        this.nodeSub = this.node$.subscribe(n => {
            if (n && !this.store.selectSnapshot(WorkflowState).hook) {
                this.editableNode = cloneDeep(n);
                delete this.editableHook;
                if (!this.editableNode.context) {
                    this.editableNode.context = new WNodeContext();
                }
                if (!this.editableNode.context.conditions) {
                    this.editableNode.context.conditions = new WorkflowNodeConditions();
                }
                if (!this.editableNode.context.conditions.plain) {
                    this.editableNode.context.conditions.plain = new Array<WorkflowNodeCondition>();
                }
                this.previousValue = this.editableNode.context.conditions.lua_script;
                let condition = this.editableNode.context.conditions.plain.find(cc => cc.variable === 'cds.manual');
                if (condition) {
                    condition.value = <any>(condition.value !== 'false');
                }
            } else {
                delete this.editableNode;
            }
            this._cd.markForCheck();
        });
        this.hookSub = this.hook$.subscribe(h => {
            if (h) {
                this.editableHook = cloneDeep(h);
                delete this.editableNode;
                if (!this.editableHook.conditions) {
                    this.editableHook.conditions = new WorkflowNodeConditions();
                }
                if (!this.editableHook.conditions.plain) {
                    this.editableHook.conditions.plain = new Array<WorkflowNodeCondition>();
                }

                this.previousValue = this.editableHook.conditions.lua_script;
                let condition = this.editableHook.conditions.plain.find(cc => cc.variable === 'cds.manual');
                if (condition) {
                    condition.value = <any>(condition.value !== 'false');
                }
            } else {
                delete this.editableHook;
            }
            this._cd.markForCheck();
        });

        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'lua',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
            readOnly: this.readonly,
        };

        this.themeSubscription = this._theme.get().pipe(finalize(() => this._cd.markForCheck())).subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });

        if (this.editableNode) {
            this._workflowService.getTriggerCondition(this.project.key, this.workflow.name, this.editableNode.id)
                .pipe(
                    first(),
                    finalize(() => {
                        this.loadingConditions = false;
                        this._cd.markForCheck();
                    })
                )
                .subscribe(wtc => this.triggerConditions = wtc);
        } else {
            this._workflowService.getTriggerHookCondition(this.project.key, this.workflow.name)
                .pipe(
                    first(),
                    finalize(() => {
                        this.loadingConditions = false;
                        this._cd.markForCheck();
                    })
                )
                .subscribe(wtc => this.triggerConditions = wtc);
        }
    }

    updateWorkflow(): void {
        this.loading = true;
        if (this.editableNode != null) {
            if (this.editableNode.context.conditions.lua_script && this.editableNode.context.conditions.lua_script !== '') {
                this.editableNode.context.conditions.plain = null;
            } else {
                this.editableNode.context.conditions.lua_script = '';
                let sizeBefore = this.editableNode.context.conditions.plain.length;


                let tmp = uniqBy(this.editableNode.context.conditions.plain, 'variable');
                let sizeAfter = tmp.length;
                if (sizeAfter !== sizeBefore) {
                    this._toast.error('Conflict', this._translate.instant('workflow_node_condition_duplicate'));
                    this.loading = false;
                    return;
                }
                this.editableNode.context.conditions.plain = tmp;

                let emptyConditions = this.editableNode.context.conditions.plain.findIndex(c => !c.variable)
                if (emptyConditions > -1) {
                    this._toast.error('Forbidden', this._translate.instant('workflow_node_condition_empty'));
                    this.loading = false;
                    return;
                }
            }
        } else if (this.editableHook != null) {
            if (this.editableHook.conditions.lua_script && this.editableHook.conditions.lua_script !== '') {
                this.editableHook.conditions.plain = null;
            } else {
                this.editableHook.conditions.lua_script = '';
                let sizeBefore = this.editableHook.conditions.plain.length;


                let tmp = uniqBy(this.editableHook.conditions.plain, 'variable');
                let sizeAfter = tmp.length;
                if (sizeAfter !== sizeBefore) {
                    this._toast.error('Conflict', this._translate.instant('workflow_node_condition_duplicate'));
                    this.loading = false;
                    return;
                }
                this.editableHook.conditions.plain = tmp;

                let emptyConditions = this.editableHook.conditions.plain.findIndex(c => !c.variable)
                if (emptyConditions > -1) {
                    this._toast.error('Forbidden', this._translate.instant('workflow_node_condition_empty'));
                    this.loading = false;
                    return;
                }
            }
        }

        let clonedWorkflow = cloneDeep(this.workflow);

        if (this.editableNode) {
            let n: WNode;
            if (this.editMode) {
                n = Workflow.getNodeByRef(this.editableNode.ref, clonedWorkflow);
            } else {
                n = Workflow.getNodeByID(this.editableNode.id, clonedWorkflow);
            }
            n.context.conditions = cloneDeep(this.editableNode.context.conditions);
            if (n.context.conditions && n.context.conditions.plain) {
                n.context.conditions.plain.forEach(cc => {
                    cc.value = cc.value.toString();
                });
            }
        } else if (this.editableHook) {
            let hook = Workflow.getHookByRef(this.editableHook.ref, clonedWorkflow);
            hook.conditions = cloneDeep(this.editableHook.conditions);
            if (hook.conditions && hook.conditions.plain) {
                hook.conditions.plain.forEach(cc => {
                    cc.value = cc.value.toString();
                });
            }
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
                this.conditionsChange.emit(false);
                if (this.editMode) {
                    this._toast.info('', this._translate.instant('workflow_ascode_updated'));
                } else {
                    this._toast.success('', this._translate.instant('workflow_updated'));
                }

            });
    }

    pushChange(event: string, e?: string): void {
        if (event !== 'codemirror') {
            this.conditionsChange.emit(true);
            return;
        }
        if (event === 'codemirror' && e && e !== this.previousValue) {
            this.previousValue = e;
            this.conditionsChange.emit(true);
        }
        return;

    }
}
