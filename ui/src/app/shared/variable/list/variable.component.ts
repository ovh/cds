import {Component, Input, EventEmitter, Output} from '@angular/core';
import {Variable} from '../../../model/variable.model';
import {SharedService} from '../../shared.service';
import {Table} from '../../table/table';
import {VariableService} from '../../../service/variable/variable.service';
import {VariableEvent} from '../variable.event.model';

@Component({
    selector: 'app-variable',
    templateUrl: './variable.html',
    styleUrls: ['./variable.scss']
})
export class VariableComponent extends Table {

    @Input() variables: Variable[];

    // display: mode
    @Input() mode = 'edit';
    @Output() event = new EventEmitter<VariableEvent>();

    private ready = false;
    private variableTypes: string[];

    constructor(private _variableService: VariableService, private _sharedService: SharedService) {
        super();
        this.variableTypes = this._variableService.getTypesFromCache();
        if (!this.variableTypes) {
            this._variableService.getTypesFromAPI().subscribe(types => {
                this.variableTypes = types;
                this.ready = true;
            });
        } else {
            this.ready = true;
        }
    }

    getData(): any[] {
        return this.variables;
    }

    /**
     * Send Event to parent component.
     * @param type Type of event (update, delete)
     * @param variable Variable data
     */
    sendEvent(type: string, variable: Variable): void {
        variable.updating = true;
        this.event.emit(new VariableEvent(type, variable));
    }

}
