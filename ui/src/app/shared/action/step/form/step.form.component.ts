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
    @Input() final: boolean;
    @Output() create = new EventEmitter<StepEvent>();

    step: Action;

    constructor() { }

    ngOnInit(): void {
        this.step = this.publicActions.find(a => a.name === 'Script');
    }

    selectPublicAction(name: string): void {
        let index = this.publicActions.findIndex(a => a.name === name);
        if (index >= 0) {
            this.step = this.publicActions[index];
        }
    };

    addStep(): void {
        if (this.step) {
            this.step.final = this.final;
            this.step.enabled = true;
            this.create.emit(new StepEvent('add', this.step));
        }
    }
}
