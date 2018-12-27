import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { cloneDeep } from 'lodash';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { PipelineStatus } from '../../../../../model/pipeline.model';
import { Workflow, WorkflowNodeCondition, WorkflowNodeConditions } from '../../../../../model/workflow.model';
declare var CodeMirror: any;

@Component({
    selector: 'app-workflow-node-condition-form',
    templateUrl: './condition.form.html',
    styleUrls: ['./condition.form.scss']
})
export class WorkflowNodeConditionFormComponent {

    @Input() operators: {};
    @Input('names')
    set names(data: string[]) {
        this._names = data;
        if (data) {
            this.suggest = data.map((d) => d.replace(/-|\./g, '_'));
        }
    }
    get names() {
        return this._names;
    }
    @Input() conditions: WorkflowNodeConditions;
    @Input() workflow: Workflow;
    @Input() mode: 'advanced'|'basic';

    @Output() changeEvent = new EventEmitter<WorkflowNodeConditions>();

    @ViewChild('textareaCodeMirror')
    codemirror: CodemirrorComponent;

    _names: Array<string> = [];
    suggest: Array<string> = [];
    condition = new WorkflowNodeCondition();
    oldVariableCondition: string;
    codeMirrorConfig: {};
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];

    constructor() {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'lua',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
    }

    send(): void {
        if (this.mode === 'basic') {
            this.conditions.plain.push(this.condition);
        }

        this.conditionsChange();
    }

    isStatusVariable(): boolean {
        return this.condition && this.condition.variable && this.condition.variable.indexOf('.status') !== -1;
    }

    conditionsChange() {
        this.changeEvent.emit(cloneDeep(this.conditions));
    }

    plainConditionsChange(plainConditions: WorkflowNodeCondition[]) {
        this.conditions.plain = plainConditions;
        this.changeEvent.emit(cloneDeep(this.conditions));
    }

    changeCodeMirror(): void {
        this.codemirror.instance.on('keyup', (cm, event) => {
            if (!cm.state.completionActive && (event.keyCode > 46 || event.keyCode === 32)) {
                CodeMirror.showHint(cm, CodeMirror.hint.condition, {
                    completeSingle: true,
                    closeCharacters: / /,
                    cdsCompletionList: this.suggest || [],
                    specialChars: ''
                });
            }
        });
    }

    updateConditionValue(event: any) {
      this.condition.value = event.target.checked ?  'true' : 'false';
    }

    variableChanged(variable: any) {
        if (!variable || variable !== this.oldVariableCondition) {
            this.condition.value = null;
            this.condition.operator = 'eq';
        }
        this.oldVariableCondition = variable;
        if (variable === 'cds.manual') {
            this.condition.value = 'false';
        }
    }
}
