import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from '@angular/core';
import { Action } from 'app/model/action.model';
import { AllKeys } from 'app/model/keys.model';
import { Parameter } from 'app/model/parameter.model';
import { StepEvent } from 'app/shared/action/step/step.event';

@Component({
    selector: 'app-action-step',
    templateUrl: './step.html',
    styleUrls: ['./step.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ActionStepComponent {

    _step: Action;
    stepURL: Array<string>;
    withAdvanced: boolean;
    collapsed_advanced = false;
    @Input()
    set step(data: Action) {
        if (data) {
            this._step = data;
            this.stepURL = ['/settings', data.group ? 'action' : 'action-builtin'];
            if (data.group) {
                this.stepURL.push(data.group.name);
            }
            this.stepURL.push(data.name);
            this._step.step_name = this._step.step_name || this._step.name;
            if (data.parameters) {
                this.withAdvanced = data.parameters.some((parameter) => parameter.advanced);
            }
        } else {
            delete this._step;
        }
    }
    get step(): Action {
        return this._step;
    }

    @Input() action: Action;
    @Input() edit: boolean;
    @Input() suggest: Array<string>;
    @Input() keys: AllKeys;
    @Input()
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

    removeStep(): void {
        this.removeEvent.emit(new StepEvent('delete', this.step));
    }
}
