import {Component, Input, Output, OnInit, EventEmitter} from '@angular/core';
import {RequirementStore} from '../../../service/worker-model/requirement/requirement.store';
import {Requirement} from '../../../model/requirement.model';
import {GroupPermission, adminGroupName} from '../../../model/group.model';
import {PermissionValue} from '../../../model/permission.model';
import {RequirementEvent} from '../requirement.event.model';
import {WorkerModelService} from '../../../service/worker-model/worker-model.service';
import {WorkerModel} from '../../../model/worker-model.model';
import {finalize, first} from 'rxjs/operators';
import {TranslateService} from '@ngx-translate/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';

@Component({
    selector: 'app-requirements-form',
    templateUrl: './requirements.form.html',
    styleUrls: ['./requirements.form.scss']
})
export class RequirementsFormComponent implements OnInit {

    @Input('suggest')
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

    get suggestWithWorkerModel() {
        return this._suggestWithWorkerModel;
    }

    @Input() modal: SemanticModalComponent;
    @Input() groupsPermission: Array<GroupPermission>;

    @Output() event = new EventEmitter<RequirementEvent>();

    newRequirement: Requirement = new Requirement('binary');
    availableRequirements: Array<string>;
    workerModels: Array<WorkerModel>;
    _suggest: Array<string> = [];
    _suggestWithWorkerModel: Array<string> = [];
    loading = true;
    workerModelLinked: WorkerModel;
    isFormValid = false;
    modelTypeClass: string;

    constructor(private _requirementStore: RequirementStore,
        private _workerModelService: WorkerModelService,
        private _translate: TranslateService) {
        this._requirementStore.getAvailableRequirements().subscribe(r => {
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
        this._workerModelService.getWorkerModels()
            .pipe(
              first(),
              finalize(() => this.loading = false)
            )
            .subscribe( wms => {
                this.workerModels = wms;
                if (Array.isArray(this.workerModels)) {
                    let filteredWm = this.workerModels;

                    if (this.groupsPermission) {
                        filteredWm = this.workerModels.filter((wm) => {
                            let groupPerm = this.groupsPermission.find((grp) => {
                                return grp.group.name === wm.group.name && grp.permission >= PermissionValue.READ_EXECUTE;
                            });

                            return groupPerm != null || wm.group.name === adminGroupName;
                        });
                    }

                    this._suggestWithWorkerModel = filteredWm.map(wm => wm.name).concat(this._suggest);
                }
            });

    }

    computeFormValid(form): void {
        this.isFormValid = (form.valid === true && this.newRequirement.name !== '' && this.newRequirement.value !== '');
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
                break
            case 'model':
                this.workerModelLinked = this.computeDisplayLinkWorkerModel();
                this.newRequirement.name = this.newRequirement.value;
                break
            case 'volume':
                this.newRequirement.name = this.getVolumeName();
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
