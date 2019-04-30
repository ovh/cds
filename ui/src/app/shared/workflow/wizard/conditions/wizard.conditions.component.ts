import {Component, Input, OnInit} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Store} from '@ngxs/store';
import {PermissionValue} from 'app/model/permission.model';
import {PipelineStatus} from 'app/model/pipeline.model';
import {Project} from 'app/model/project.model';
import {WNode, WNodeContext, Workflow, WorkflowNodeCondition, WorkflowNodeConditions} from 'app/model/workflow.model';
import {VariableService} from 'app/service/variable/variable.service';
import {WorkflowService} from 'app/service/workflow/workflow.service';
import {Table} from 'app/shared/table/table';
import {ToastService} from 'app/shared/toast/ToastService';
import {UpdateWorkflow} from 'app/store/workflows.action';
import {cloneDeep} from 'lodash';
import {finalize, first} from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-conditions',
    templateUrl: './wizard.conditions.html',
    styleUrls: ['./wizard.conditions.scss']
})
export class WorkflowWizardNodeConditionComponent extends Table<WorkflowNodeCondition> implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;
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
            if (this.editableNode.context.conditions.lua_script && this.editableNode.context.conditions.lua_script !== '') {
                this.isAdvanced = true;
            } else {
                this.isAdvanced = false;
            }

            let c = this.editableNode.context.conditions.plain.find(cc => cc.variable === 'cds.manual');
            if (c) {
                c.value = <any>(c.value !== 'false');
            }

        }
    };
    get node(): WNode {
        return this.editableNode;
    }

    codeMirrorConfig: {};
    isAdvanced = false;
    suggest: Array<string> = [];
    loadingConditions = false;
    operators: {};
    conditionNames: Array<string>;
    permission = PermissionValue;
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];
    loading = false;

    constructor(private store: Store, private _variableService: VariableService, private _workflowService: WorkflowService,
                private _toast: ToastService, private _translate: TranslateService) {
        super();
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'lua',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
            readOnly: true,
        };
    }

    getData(): Array<WorkflowNodeCondition> {
        return undefined;
    }

    ngOnInit(): void {
        this._variableService.getContextVariable(this.project.key, this.node.context.pipeline_id)
            .subscribe((suggest) => this.suggest = suggest);

        this._workflowService.getTriggerCondition(this.project.key, this.workflow.name, this.node.id)
            .pipe(
                first(),
                finalize(() => this.loadingConditions = false)
            )
            .subscribe(wtc => {
                this.operators = wtc.operators;
                this.conditionNames = wtc.names;
            });
    }

    removeCondition(index: number): void {
        this.editableNode.context.conditions.plain.splice(index, 1);
    }

    addEmptyCondition(): void {
        let emptyCond = new WorkflowNodeCondition();
        emptyCond.operator = 'eq';
        this.editableNode.context.conditions.plain.push(emptyCond);
    }

    updateWorkflow(): void {
        this.loading = true;
        if (this.isAdvanced) {
            this.editableNode.context.conditions.plain = null;
        } else {
            this.editableNode.context.conditions.lua_script = '';
            let sizeBefore = this.editableNode.context.conditions.plain.length;
            let tmp = this.getUnique(this.editableNode.context.conditions.plain, 'variable');
            let sizeAfter = tmp.length;
            if (sizeAfter !== sizeBefore) {
                this._toast.error('Conflict', this._translate.instant('workflow_node_condition_duplicate'));
                this.loading = false;
                return;
            }
            this.editableNode.context.conditions.plain = tmp;

            let emptyConditions = this.editableNode.context.conditions.plain.findIndex(c => (!c.variable || c.variable === ''))
            if (emptyConditions > -1) {
                this._toast.error('Forbidden', this._translate.instant('workflow_node_condition_empty'));
                this.loading = false;
                return;
            }
        }

        let clonedWorkflow = cloneDeep(this.workflow);
        let n = Workflow.getNodeByID(this.editableNode.id, clonedWorkflow);
        n.context.conditions = cloneDeep(this.editableNode.context.conditions);
        n.context.conditions.plain.forEach(cc => {
            cc.value = cc.value.toString();
        });
        this.store.dispatch(new UpdateWorkflow({
            projectKey: this.workflow.project_key,
            workflowName: this.workflow.name,
            changes: clonedWorkflow
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }

    getUnique(arr, comp): Array<WorkflowNodeCondition> {
        const unique = arr
            .map(e => e[comp])
            // store the keys of the unique objects
            .map((e, i, final) => final.indexOf(e) === i && i)
            // eliminate the dead keys & store unique objects
            .filter(e => arr[e]).map(e => arr[e]);
        return unique;
    }
}
