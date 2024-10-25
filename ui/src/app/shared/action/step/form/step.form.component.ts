import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output
} from '@angular/core';
import { Action } from 'app/model/action.model';
import { StepEvent } from 'app/shared/action/step/step.event';

@Component({
    selector: 'app-action-step-form',
    templateUrl: './step.form.html',
    styleUrls: ['./step.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ActionStepFormComponent {

    _actions: Array<Action>;
    @Input() set actions(as: Array<Action>) {
        this._actions = as;
        this.initDefaultActionSelected();
    }
    get actions(): Array<Action> {
        return this._actions;
    }

    @Output() onEvent = new EventEmitter<StepEvent>();

    expended: boolean;
    selected: Action;

    constructor(private _cd: ChangeDetectorRef) {
    }

    initDefaultActionSelected(): void {
        let script = this._actions?.find(a => a.name === 'Script' && a.type === 'Builtin');
        if (script && !this.selected) {
            this.selected = script;
            this._cd.markForCheck();
        }
    }

    selectAction(id: number): void {
        this.selected = this.actions.find(a => a.id === Number(id));
    }

    clickAddStep(): void {
        this.expended = false;
        this.onEvent.emit(new StepEvent('add', this.selected));
    }

    clickCancel(): void {
        this.expended = false;
        this.onEvent.emit(new StepEvent('cancel', null));
    }

    showActions(): void {
        this.expended = true;
        this.onEvent.emit(new StepEvent('expend', null));
    }
}
