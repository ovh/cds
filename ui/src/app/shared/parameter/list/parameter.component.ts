import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output
} from '@angular/core';
import { AllKeys } from 'app/model/keys.model';
import { Parameter } from 'app/model/parameter.model';
import { Project } from 'app/model/project.model';
import { ParameterService } from 'app/service/parameter/parameter.service';
import { ParameterEvent } from 'app/shared/parameter/parameter.event.model';
import { SharedService } from 'app/shared/shared.service';
import { Table } from 'app/shared/table/table';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-parameter-list',
    templateUrl: './parameter.html',
    styleUrls: ['./parameter.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ParameterListComponent extends Table<Parameter> implements OnInit {
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

        if (this.ready) {
            this.getDataForCurrentPage();
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
    public data: Array<any> = [];

    private _parameters: Array<Parameter>;

    constructor(
        private _paramService: ParameterService,
        public _sharedService: SharedService,
        private _cd: ChangeDetectorRef
    ) {
        super();
        this.parameterTypes = this._paramService.getTypesFromCache();
        if (!this.parameterTypes) {
            this._paramService.getTypesFromAPI().pipe(finalize(() => {
                this.ready = true;
                this._cd.markForCheck()
            })).subscribe(types => {
                this.parameterTypes = types;
            });
        } else {
            this.ready = true;
        }
    }

    ngOnInit() {
        this.getDataForCurrentPage();
    }

    getDataForCurrentPage(): any[] {
        if (this.mode === 'job') {
            this.data = this.getData();
            return this.data;
        }
        this.data = super.getDataForCurrentPage();

        return this.data;
    }

    getData(): Array<Parameter> {
        if (!this.parameters) {
            return [];
        }

        return this.parameters.map((p) => {
            p.ref = this.getRef(p);
            return p;
        });
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
