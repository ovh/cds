import { Component, EventEmitter, Input, OnInit, Output, ViewChild } from '@angular/core';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Workflow, WorkflowNodeCondition, WorkflowNodeConditions } from 'app/model/workflow.model';
import { ThemeStore } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { cloneDeep } from 'lodash';
import { Subscription } from 'rxjs/Subscription';
declare var CodeMirror: any;

@Component({
    selector: 'app-workflow-node-condition-form',
    templateUrl: './condition.form.html',
    styleUrls: ['./condition.form.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeConditionFormComponent implements OnInit {

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
    @Input() mode: 'advanced' | 'basic';

    @Output() changeEvent = new EventEmitter<WorkflowNodeConditions>();

    @ViewChild('textareaCodeMirror') codemirror: any;

    _names: Array<string> = [];
    suggest: Array<string> = [];
    condition = new WorkflowNodeCondition();
    oldVariableCondition: string;
    codeMirrorConfig: any;
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];
    themeSubscription: Subscription;

    constructor(
        private _theme: ThemeStore
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'lua',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
    }

    ngOnInit(): void {
        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
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
        this.condition.value = event.target.checked ? 'true' : 'false';
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
