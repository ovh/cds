import {Component, ElementRef, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {AllKeys} from '../../../model/keys.model';
import {Parameter} from '../../../model/parameter.model';
import {Project} from '../../../model/project.model';
import {ParameterService} from '../../../service/parameter/parameter.service';
import {SharedService} from '../../shared.service';
import {ParameterEvent} from '../parameter.event.model';

@Component({
    selector: 'app-parameter-form',
    templateUrl: './parameter.form.html',
    styleUrls: ['./parameter.form.scss']
})
export class ParameterFormComponent {

    parameterTypes: string[];
    newParameter: Parameter;

    @Input() project: Project;
    @Input() suggest: Array<string>;
    @Input() keys: AllKeys;
    @ViewChild('selectType') selectType: ElementRef;

    @Output() createParameterEvent = new EventEmitter<ParameterEvent>();

    constructor(private _paramService: ParameterService, public _sharedService: SharedService) {
        this.newParameter = new Parameter();
        this.parameterTypes = this._paramService.getTypesFromCache();
        if (!this.parameterTypes) {
            this._paramService.getTypesFromAPI().subscribe(types => {
                this.parameterTypes = types;
                this.newParameter.type = types[0];
            });
        } else {
            this.newParameter.type = this.parameterTypes[0];
        }
    }

    create(): void {
        let previousType = this.newParameter.type;
        let event: ParameterEvent = new ParameterEvent('add', this.newParameter);
        if (!this.newParameter.value) {
            switch (this.newParameter.type) {
                case 'number':
                    this.newParameter.value = '0';
                    break;
                case 'boolean':
                    this.newParameter.value = 'false';
                    break;
                default:
                    this.newParameter.value = '';
            }
        }
        this.createParameterEvent.emit(event);
        this.newParameter = new Parameter();
        this.newParameter.type = previousType;
    }

}
