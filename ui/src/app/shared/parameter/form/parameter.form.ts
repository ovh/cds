import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    Output
} from '@angular/core';
import { AllKeys } from 'app/model/keys.model';
import { Parameter } from 'app/model/parameter.model';
import { Project } from 'app/model/project.model';
import { ParameterService } from 'app/service/parameter/parameter.service';
import { ParameterEvent } from 'app/shared/parameter/parameter.event.model';
import { SharedService } from 'app/shared/shared.service';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-parameter-form',
    templateUrl: './parameter.form.html',
    styleUrls: ['./parameter.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ParameterFormComponent {

    @Input() project: Project;
    @Input() suggest: Array<string>;
    @Input() keys: AllKeys;
    @Output() createParameterEvent = new EventEmitter<ParameterEvent>();

    parameterTypes: string[];
    newParameter: Parameter;

    constructor(
        private _paramService: ParameterService,
        public _sharedService: SharedService, // used in html
        private _cd: ChangeDetectorRef
    ) {
        this.newParameter = new Parameter();
        this.parameterTypes = this._paramService.getTypesFromCache();
        if (!this.parameterTypes) {
            this._paramService.getTypesFromAPI().pipe(finalize(() => this._cd.markForCheck())).subscribe(types => {
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
