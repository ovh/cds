import {Component, Output, EventEmitter, Input} from '@angular/core';
import {Variable} from '../../../model/variable.model';
import {VariableService} from '../../../service/variable/variable.service';
import {VariableEvent} from '../variable.event.model';

@Component({
    selector: 'app-variable-form',
    templateUrl: './variable.form.html',
    styleUrls: ['./variable.form.scss']
})
export class VariableFormComponent {


    public variableTypes: string[];
    newVariable = new Variable();

    @Input() loading = false;
    @Output() createVariableEvent = new EventEmitter<VariableEvent>();

    constructor(private _variableService: VariableService) {
        this.variableTypes = this._variableService.getTypesFromCache();
        if (!this.variableTypes) {
            this._variableService.getTypesFromAPI().subscribe(types => this.variableTypes = types);
        }
    }

    create(): void {
        let event: VariableEvent = new VariableEvent('add', this.newVariable);
        this.createVariableEvent.emit(event);
        this.newVariable = new Variable();
    }

}
