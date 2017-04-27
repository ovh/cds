import {Component, Input, EventEmitter, Output} from '@angular/core';
import {Action} from '../../../model/action.model';
import {StepEvent} from './step.event';

@Component({
    selector: 'app-action-step',
    templateUrl: './step.html',
    styleUrls: ['./step.scss']
})
export class ActionStepComponent {

    @Input() action: Action;
    @Input() step: Action;
    @Input() edit: boolean;
    @Input() suggest: Array<string>;

    @Output() removeEvent = new EventEmitter<StepEvent>();

    constructor() { }

    updateStepBool(b: boolean): boolean {
        this.action.hasChanged = true;
        return !b;
    }

    removeStep(): void {
        this.removeEvent.emit(new StepEvent('delete', this.step));
    }
}
