import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { SharedInfraGroupName } from 'app/model/group.model';
import { Requirement } from 'app/model/requirement.model';
import { WorkerModel } from 'app/model/worker-model.model';
import { RequirementStore } from 'app/service/requirement/requirement.store';
import { RequirementEvent } from 'app/shared/requirements/requirement.event.model';
import { SemanticModalComponent } from 'ng-semantic/ng-semantic';
import { finalize, first } from 'rxjs/operators';

export const OSArchitecture = 'os-architecture';

@Component({
    selector: 'app-requirements-form',
    templateUrl: './requirements.form.html',
    styleUrls: ['./requirements.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RequirementsFormComponent implements OnInit {
    @Input()
    set suggest(data: Array<string>) {
        if (data) {
            this._suggest = data;
        } else {
            this._suggest = [];
        }
    }
    get suggest() {
 return this._suggest;
}

    _workerModels: Array<WorkerModel>;
    @Input() set workerModels(wms: Array<WorkerModel>) {
        if (wms) {
            this._workerModels = wms;

            this.suggestWithWorkerModel = wms.map(wm => {
                if (wm.group.name !== SharedInfraGroupName) {
                    return `${wm.group.name}/${wm.name}`;
                }
                return wm.name;
            }).concat(this._suggest);
        }
    }
    get workerModels() {
 return this._workerModels;
}


    @Input() modal: SemanticModalComponent;
    @Input() config: { disableModel?: boolean, disableHostname?: boolean, disableRegion?: boolean };

    @Output() event = new EventEmitter<RequirementEvent>();

    newRequirement: Requirement = new Requirement('binary');
    availableRequirements: Array<string>;
    _suggest: Array<string> = [];
    suggestWithWorkerModel: Array<string> = [];
    suggestWithOsArch: Array<string> = [];
    workerModelLinked: WorkerModel;
    isFormValid = false;
    modelTypeClass: string;
    popupText: string;
    placeholderTypeName: {};
    placeholderTypeValue: {};

    constructor(
        private _requirementStore: RequirementStore,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef
    ) {
        this._requirementStore.getAvailableRequirements().pipe(finalize( () => this._cd.markForCheck())).subscribe(r => {
            this.availableRequirements = new Array<string>();
            this.placeholderTypeName = {};
            this.placeholderTypeValue = {};

            // user does not need to add plugin prerequisite manually, so we remove it from list
            this.availableRequirements.push(...r.filter(req => req !== 'plugin').toArray());

            this.availableRequirements.forEach(a => {
                let placeHolderName = '';
                let placeHolderValue = '';
                switch (a) {
                    case 'binary':
                        placeHolderValue = 'bash';
                        break;
                    case 'service':
                        placeHolderName = this._translate.instant('requirement_placeholder_name_service');
                        placeHolderValue = 'postgres:9.5.3';
                        break;
                    case 'hostname':
                        placeHolderValue = this._translate.instant('requirement_placeholder_value_hostname');
                        break;
                    case 'memory':
                        placeHolderValue = '4096';
                        break;
                    case 'os-architecture':
                        placeHolderName = this._translate.instant('requirement_placeholder_name_os-architecture');
                        placeHolderValue = 'linux-amd64';
                        break;
                    case 'model':
                        break;
                }
                this.placeholderTypeName[a] = placeHolderName;
                this.placeholderTypeValue[a] = placeHolderValue;
            });
        });
    }

    onSubmitAddRequirement(form): void {
        this.computeFormValid(form);
        if (this.isFormValid) {
            this.event.emit(new RequirementEvent('add', this.newRequirement));
            this.newRequirement = new Requirement('binary');
        }
    }

    ngOnInit() {
        this._requirementStore.getRequirementsTypeValues(OSArchitecture).pipe(first()).subscribe(values => {
            this.suggestWithOsArch = values.concat(this._suggest);
        });
    }

    computeFormValid(form): void {
        this.popupText = '';
        let goodModel = this.newRequirement.type !== 'model' || !this.config.disableModel;
        let goodHostname = this.newRequirement.type !== 'hostname' || !this.config.disableHostname;
        let goodRegion = this.newRequirement.type !== 'region' || !this.config.disableRegion;
        this.isFormValid = (form.valid === true && this.newRequirement.name !== '' && this.newRequirement.value !== '')
            && goodModel && goodHostname && goodRegion;
        if (!goodModel) {
            this.popupText = this._translate.instant('requirement_error_model');
        }
        if (!goodHostname) {
            this.popupText = this._translate.instant('requirement_error_hostname');
        }
        if (!goodRegion) {
            this.popupText = this._translate.instant('requirement_error_region');
        }
    }

    selectType(): void {
        this.newRequirement.value = '';
        this.newRequirement.opts = '';
        this.newRequirement.name = '';
    }

    setName(form): void {
        switch (this.newRequirement.type) {
            case 'service':
                // if type service, user have to choose a hostname
                break;
            case 'memory':
                // memory: memory_4096
                this.newRequirement.name = 'memory_' + this.newRequirement.value;
                break;
            case 'model':
                this.workerModelLinked = this.computeDisplayLinkWorkerModel();
                this.newRequirement.name = this.newRequirement.value;
                break;
            case OSArchitecture:
                this.newRequirement.name = OSArchitecture;
                break;
            default:
                // else, name is the value of the requirement
                this.newRequirement.name = this.newRequirement.value;
        }
        this.computeFormValid(form);
    }

    closeModal() {
        if (this.modal) {
            this.modal.hide();
        }
    }

    computeDisplayLinkWorkerModel(): WorkerModel {
        if (this.newRequirement.value === '' || !Array.isArray(this.workerModels)) {
            return null;
        }

        return this.workerModels.find((wm) => wm.name === this.newRequirement.value);
    }
}
