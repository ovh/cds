import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import {Action} from '../../../model/action.model';
import {StepEvent} from './step.event';
import {Parameter} from '../../../model/parameter.model';

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
    @Input('publicActions')
    set publicActions(data: Array<Action>){
        if (data) {
            let originalAction = data.find(a => a.name === this.step.name);
            if (originalAction && originalAction.parameters) {
                this.originalParam = new Map<string, Parameter>();
                originalAction.parameters.forEach(p => {
                    this.originalParam.set(p.name, p);
                });
            }
        }
    }

    @Output() removeEvent = new EventEmitter<StepEvent>();

    originalParam = new Map<string, Parameter>();

    constructor() { }

    updateStepBool(b: boolean): boolean {
        this.action.hasChanged = true;
        return !b;
    }

    removeStep(): void {
        this.removeEvent.emit(new StepEvent('delete', this.step));
    }
}
