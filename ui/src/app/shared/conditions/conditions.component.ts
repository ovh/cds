import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnInit, Output, ViewChild } from '@angular/core';
import { PermissionValue } from 'app/model/permission.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WorkflowNodeCondition, WorkflowNodeConditions, WorkflowTriggerConditionCache } from 'app/model/workflow.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Table } from 'app/shared/table/table';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { Subscription } from 'rxjs/Subscription';

declare var CodeMirror: any;

@Component({
    selector: 'app-conditions',
    templateUrl: './conditions.html',
    styleUrls: ['./conditions.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ConditionsComponent extends Table<WorkflowNodeCondition> implements OnInit {
    @Input('triggerConditions') set triggerConditions(data: WorkflowTriggerConditionCache) {
        this._triggerCondition = data;
        if (data) {
            this.operators = Object.keys(data.operators).map(k => {
                return { key: k, value: data.operators[k] };
            });
            this.conditionNames = data.names;
            if (this.conditionNames) {
                this.suggest = this.conditionNames.map((d) => d.replace(/-|\./g, '_'));
            }
        }
    }
    get triggerConditions(): WorkflowTriggerConditionCache {
        return this._triggerCondition;
    }
    @Input('conditions') set conditions(conditions: WorkflowNodeConditions) {
        this._conditions = conditions;
        if (this._conditions.lua_script && this._conditions.lua_script !== '') {
            this.isAdvanced = true;
        } else {
            this.isAdvanced = false;
        }
    }
    get conditions(): WorkflowNodeConditions {
        return this._conditions;
    }

    @Input() project: Project;
    @Input() pipelineId: number;

    _conditions: WorkflowNodeConditions;
    @Input() readonly = true;

    @Output() conditionsChange = new EventEmitter<WorkflowNodeConditions>();

    @ViewChild('textareaCodeMirror', {static: false}) codemirror: CodemirrorComponent;
    codeMirrorConfig: any;
    isAdvanced = false;
    suggest: Array<string> = [];
    loadingConditions = false;
    operators: Array<any>;
    conditionNames: Array<string>;
    permission = PermissionValue;
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];
    loading = false;
    previousValue: string;
    themeSubscription: Subscription;

    _triggerCondition: WorkflowTriggerConditionCache;

    constructor(
        private _theme: ThemeStore
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

        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });

        if (!this.conditions) {
            this.conditions = new WorkflowNodeConditions();
        }
        if (!this.conditions.plain) {
            this.conditions.plain = new Array<WorkflowNodeCondition>();
        }

        this.previousValue = this.conditions.lua_script;
        let condition = this.conditions.plain.find(cc => cc.variable === 'cds.manual');
        if (condition) {
            condition.value = <any>(condition.value !== 'false');
        }
    }

    removeCondition(index: number): void {
        this.conditions.plain.splice(index, 1);
        this.pushChange('remove');
    }

    addEmptyCondition(): void {
        let emptyCond = new WorkflowNodeCondition();
        emptyCond.operator = 'eq';

        if (!this.conditions.plain) {
            this.conditions.plain = [emptyCond];
        } else {
            this.conditions.plain.push(emptyCond);
        }
        this.conditionsChange.emit(this.conditions);
    }

    pushChange(event: string, e?: string): void {
        if (event !== 'codemirror') {
            this.conditionsChange.emit(this.conditions);
            return;
        }
        if (event === 'codemirror' && e && e !== this.previousValue) {
            this.previousValue = e;
            this.conditionsChange.emit(this.conditions);
        }
        return;

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
}
