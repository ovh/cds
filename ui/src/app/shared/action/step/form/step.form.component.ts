import {Component, Input, Output, EventEmitter, OnInit } from '@angular/core';
import {Action} from '../../../../model/action.model';
import {StepEvent} from '../step.event';

@Component({
    selector: 'app-action-step-form',
    templateUrl: './step.form.html',
    styleUrls: ['./step.form.scss']
})
export class ActionStepFormComponent implements OnInit {
    @Input() publicActions: Array<Action>;
    showAddStep: boolean;
    @Output() create = new EventEmitter<StepEvent>();

    step: Action;

    constructor() { }

    ngOnInit(): void {
        this.step = this.publicActions.find(a => a.name === 'Script');
    }

    selectPublicAction(name: string): void {
        let step = this.publicActions.find(a => a.name === name);
        if (step) {
            this.step = step;
        }
    };

    addStep(optional: boolean, always_executed: boolean): void {
        if (this.step) {
            this.step.optional = optional;
            this.step.always_executed = always_executed;
            this.step.enabled = true;
            this.showAddStep = false;
            this.create.emit(new StepEvent('add', this.step));
        }
    }

    cancel(): void {
        this.showAddStep = false;
        this.create.emit(new StepEvent('cancel', null));
    }

    displayChoice(): void {
        this.showAddStep = true;
        this.create.emit(new StepEvent('displayChoice', null));
    }
}
