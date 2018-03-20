import {Component, Input, EventEmitter, Output} from '@angular/core';
import {SharedService} from '../../shared.service';
import {Table} from '../../table/table';
import {Parameter} from '../../../model/parameter.model';
import {ParameterEvent} from '../parameter.event.model';
import {ParameterService} from '../../../service/parameter/parameter.service';
import {Project} from '../../../model/project.model';
import {AllKeys} from '../../../model/keys.model';

@Component({
    selector: 'app-parameter-list',
    templateUrl: './parameter.html',
    styleUrls: ['./parameter.scss']
})
export class ParameterListComponent extends Table {

    @Input('parameters')
    set parameters(newP: Array<Parameter>) {
        if (Array.isArray(newP)) {
            this._parameters = newP.map((d) => {
                d.previousName = d.name;
                return d;
            });
        } else {
            this._parameters = newP;
        }
    }
    get parameters() {
        return this._parameters;
    }
    @Input() paramsRef: Array<Parameter>;
    @Input() project: Project;
    @Input() suggest: Array<string>;
    @Input() keys: AllKeys;
    @Input() canDelete: boolean;
    @Input() hideSave = false;

    // edit/launcher/ro/job
    @Input() mode = 'edit';
    @Output() event = new EventEmitter<ParameterEvent>();

    public ready = false;
    public parameterTypes: string[];

    private _parameters: Array<Parameter>;

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

    getDataForCurrentPage(): any[] {
        if (this.mode === 'job') {
            return this.getData();
        }
        return super.getDataForCurrentPage();
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

    getColspan(): number {
        if (this.mode && this.mode === 'launcher') {
            if (this.canDelete) {
                return 3;
            }
            return 2;
        }
        return 4;
    }

    getRef(p: Parameter): Parameter {
        if (this.paramsRef) {
            return this.paramsRef.find(r => r.name === p.name);
        }
        return null;
    }

}
