import {Component, Input, EventEmitter, Output} from '@angular/core';
import {SharedService} from '../../shared.service';
import {Table} from '../../table/table';
import {Parameter} from '../../../model/parameter.model';
import {ParameterEvent} from '../parameter.event.model';
import {ParameterService} from '../../../service/parameter/parameter.service';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-parameter-list',
    templateUrl: './parameter.html',
    styleUrls: ['./parameter.scss']
})
export class ParameterListComponent extends Table {

    @Input() parameters: Array<Parameter>;
    @Input() project: Project;

    // edit/launcher/ro
    @Input() mode = 'edit';
    @Output() event = new EventEmitter<ParameterEvent>();

    public ready = false;
    public parameterTypes: string[];

    constructor(private _paramService: ParameterService, public _sharedService: SharedService) {
        super();
        this.parameterTypes = this._paramService.getTypesFromCache();
        if (!this.parameterTypes) {
            this._paramService.getTypesFromAPI().subscribe(types => {
                this.parameterTypes = types;
                this.ready = true;
            });
        } else {
            this.ready = true;
        }
    }

    getData(): any[] {
        return this.parameters;
    }

    /**
     * Send Event to parent component.
     * @param type Type of event (delete)
     * @param param parameter data
     */
    sendEvent(type: string, param: Parameter): void {
        this.event.emit(new ParameterEvent(type, param));
    }

}
