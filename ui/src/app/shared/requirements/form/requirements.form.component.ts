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
    get suggest() { return this._suggest; }

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
    get workerModels() { return this._workerModels; }


    @Input() modal: SemanticModalComponent;
    @Input() config: { disableModel?: boolean, disableHostname?: boolean };

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

    constructor(
        private _requirementStore: RequirementStore,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef,
    ) {
        this._requirementStore.getAvailableRequirements().pipe(finalize( () => this._cd.markForCheck())).subscribe(r => {
            this.availableRequirements = new Array<string>();
            // user does not need to add plugin prequisite manually, so we remove it from list
            this.availableRequirements.push(...r.filter(req => req !== 'plugin').toArray());
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
        this.isFormValid = (form.valid === true && this.newRequirement.name !== '' && this.newRequirement.value !== '')
            && goodModel && goodHostname;
        if (!goodModel) {
            this.popupText = this._translate.instant('requirement_error_model');
        }
        if (!goodHostname) {
            this.popupText = this._translate.instant('requirement_error_hostname');
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
            case 'volume':
                this.newRequirement.name = this.getVolumeName();
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

    getVolumeName(): string {
        let parts = this.newRequirement.value.split(',');
        for (let p of parts) {
            // example: type=bind,source=/hostDir/sourceDir,destination=/dirInJob
            // we want /dirInJob for volume name
            if (p.startsWith('destination=')) {
                let value = p.split('=');
                if (value.length === 2) {
                    // keep only a-zA-Z - and / in name, '_' for others characters
                    return value[1].replace(/([^a-zA-Z\-/])/gi, '_');
                }
            }
        }
        return '';
    }

    getHelp() {
        return this._translate.instant('requirement_help_' + this.newRequirement.type);
    }

    computeDisplayLinkWorkerModel(): WorkerModel {
        if (this.newRequirement.value === '' || !Array.isArray(this.workerModels)) {
            return null;
        }

        return this.workerModels.find((wm) => wm.name === this.newRequirement.value);
    }
}
