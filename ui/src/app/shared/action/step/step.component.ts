import { Component, EventEmitter, Input, Output } from '@angular/core';
import {Action} from '../../../model/action.model';
import {AllKeys} from '../../../model/keys.model';
import {Parameter} from '../../../model/parameter.model';
import {StepEvent} from './step.event';

@Component({
    selector: 'app-action-step',
    templateUrl: './step.html',
    styleUrls: ['./step.scss']
})
export class ActionStepComponent {

    _step: Action;
    withAdvanced: boolean;
    @Input('step')
    set step(step: Action) {
        this._step = step;
        if (step) {
            this.withAdvanced = step.parameters.some((parameter) => parameter.advanced);
        }
    }
    get step(): Action {
        return this._step;
    }

    @Input() action: Action;
    @Input() edit: boolean;
    @Input() suggest: Array<string>;
    @Input() keys: AllKeys;
    @Input('publicActions')
    set publicActions(data: Array<Action>) {
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
    @Input() collapsed;
    @Input() order;

    @Output() removeEvent = new EventEmitter<StepEvent>();

    originalParam = new Map<string, Parameter>();
    constructor() {
         this.collapsed = true;
    }
    updateStepBool(b: boolean): boolean {
        this.action.hasChanged = true;
        return !b;
    }

    removeStep(): void {
        this.removeEvent.emit(new StepEvent('delete', this.step));
    }
}
