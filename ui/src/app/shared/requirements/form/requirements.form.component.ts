import {Component, Input, Output, EventEmitter, ViewChild} from '@angular/core';
import {RequirementStore} from '../../../service/worker-model/requirement/requirement.store';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';
import {WorkerModelService} from '../../../service/worker-model/worker-model.service';
import {WorkerModel} from '../../../model/worker-model.model';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-requirements-form',
    templateUrl: './requirements.form.html',
    styleUrls: ['./requirements.form.scss']
})
export class RequirementsFormComponent {

    @Input('suggest')
    set suggest(data: Array<string>) {
        if (Array.isArray(this.workerModels) && data) {
            this.workerModels = this.workerModels.concat(data);
        } else if (data) {
            this.workerModels = data;
        }
        this._suggest = data || [];
    }
    get suggest() {
        return this._suggest;
    }
    @Output() event = new EventEmitter<RequirementEvent>();

    newRequirement: Requirement = new Requirement('binary');
    availableRequirements: Array<string>;
    valueChanged = false;
    workerModels: Array<string>;
    _suggest: Array<string> = [];
    loading = true;

    constructor(private _requirementStore: RequirementStore, private _workerModelService: WorkerModelService) {
        this._requirementStore.getAvailableRequirements().subscribe(r => {
            this.availableRequirements = new Array<string>();
            this.availableRequirements.push(...r.toArray());
        });

        this._workerModelService.getWorkerModels().first()
        .pipe(finalize(() => this.loading = false))
        .subscribe( wms => {
            this.workerModels = wms.map((wm) => wm.name).concat(this.workerModels);
        });
    }

    addRequirement(): void {
        this.event.emit(new RequirementEvent('add', this.newRequirement));
        this.newRequirement = new Requirement('binary');
        this.valueChanged = false;
    }

    setValue(event: any): void  {
        if (!this.valueChanged || this.newRequirement.value === '') {
            this.newRequirement.value = event.target.value;
        }
    }

    setName(event: any): void {
        this.valueChanged = true;
        if (this.newRequirement.name === '') {
            this.newRequirement.name = event.target.value;
        }
    }
}
