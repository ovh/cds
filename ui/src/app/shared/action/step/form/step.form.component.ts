import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { Action } from 'app/model/action.model';
import { StepEvent } from 'app/shared/action/step/step.event';

@Component({
    selector: 'app-action-step-form',
    templateUrl: './step.form.html',
    styleUrls: ['./step.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ActionStepFormComponent implements OnInit {
    @Input() actions: Array<Action>;
    @Output() onEvent = new EventEmitter<StepEvent>();

    expended: boolean;
    selectedID: number;
    selected: Action;

    ngOnInit(): void {
        let script = this.actions.find(a => a.name === 'Script' && a.type === 'Builtin');
        if (script) {
            this.selectedID = script.id;
            this.selected = script;
        }
    }

    selectAction(id: number): void {
        this.selected = this.actions.find(a => a.id === Number(id));
    };

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
