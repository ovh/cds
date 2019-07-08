import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output,
    ViewChild
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { PermissionValue } from 'app/model/permission.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import {
    WNode,
    WNodeContext,
    Workflow,
    WorkflowNodeCondition,
    WorkflowNodeConditions,
    WorkflowTriggerConditionCache
} from 'app/model/workflow.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { VariableService } from 'app/service/variable/variable.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Table } from 'app/shared/table/table';
import { ToastService } from 'app/shared/toast/ToastService';
import { UpdateWorkflow } from 'app/store/workflow.action';
import cloneDeep from 'lodash-es/cloneDeep';
import uniqBy from 'lodash-es/uniqBy';
import { finalize, first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-node-conditions',
    templateUrl: './wizard.conditions.html',
    styleUrls: ['./wizard.conditions.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWizardNodeConditionComponent extends Table<WorkflowNodeCondition> implements OnInit {
    @ViewChild('textareaCodeMirror', {static: false}) codemirror: any;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() pipelineId: number;
    editableNode: WNode;
    @Input('node') set node(data: WNode) {
        if (data) {
            this.editableNode = cloneDeep(data);
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
        }
    };
    get node(): WNode {
        return this.editableNode;
    }
    @Input() readonly = true;

    @Output() conditionsChange = new EventEmitter<boolean>();

    codeMirrorConfig: any;
    suggest: Array<string> = [];
    loadingConditions = false;
    operators: Array<any>;
    conditionNames: Array<string>;
    permission = PermissionValue;
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];
    loading = false;
    previousValue: string;
    themeSubscription: Subscription;
    triggerConditions: WorkflowTriggerConditionCache;

    constructor(
        private store: Store,
        private _variableService: VariableService,
        private _workflowService: WorkflowService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        super();
    }

    getData(): Array<WorkflowNodeCondition> {
        return undefined;
    }

    ngOnInit(): void {
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

        this._variableService.getContextVariable(this.project.key, this.node.context.pipeline_id)
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe((suggest) => this.suggest = suggest);

        this._workflowService.getTriggerCondition(this.project.key, this.workflow.name, this.node.id)
            .pipe(
                first(),
                finalize(() => {
                    this.loadingConditions = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe(wtc => {
                this.triggerConditions = wtc;
                this.operators = Object.keys(wtc.operators).map(k => {
                    return { key: k, value: wtc.operators[k] };
                });
                this.conditionNames = wtc.names;
            });
    }

    updateWorkflow(): void {
        this.loading = true;
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

        let clonedWorkflow = cloneDeep(this.workflow);
        let n = Workflow.getNodeByID(this.editableNode.id, clonedWorkflow);
        n.context.conditions = cloneDeep(this.editableNode.context.conditions);
        if (n.context.conditions && n.context.conditions.plain) {
            n.context.conditions.plain.forEach(cc => {
                cc.value = cc.value.toString();
            });
        }

        this.store.dispatch(new UpdateWorkflow({
            projectKey: this.workflow.project_key,
            workflowName: this.workflow.name,
            changes: clonedWorkflow
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.conditionsChange.emit(false);
                this._toast.success('', this._translate.instant('workflow_updated'));
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
