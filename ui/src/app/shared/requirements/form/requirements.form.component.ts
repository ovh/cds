import {Component, Input, Output, EventEmitter} from '@angular/core';
import {RequirementStore} from '../../../service/worker-model/requirement/requirement.store';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';
import {WorkerModelService} from '../../../service/worker-model/worker-model.service';
import {WorkerModel} from '../../../model/worker-model.model';

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
    }
    @Output() event = new EventEmitter<RequirementEvent>();

    newRequirement: Requirement = new Requirement('binary');
    availableRequirements: Array<string>;
    isWriting = false;
    workerModels: Array<string>;

    constructor(private _requirementStore: RequirementStore, private _workerModelService: WorkerModelService) {
        this._requirementStore.getAvailableRequirements().subscribe(r => {
            this.availableRequirements = new Array<string>();
            this.availableRequirements.push(...r.toArray());
        });

        this._workerModelService.getWorkerModels().first().subscribe( wms => {
            this.workerModels = wms.map((wm) => wm.name).concat(this.workerModels);
        });
    }

    addRequirement(): void {
        this.event.emit(new RequirementEvent('add', this.newRequirement));
        this.newRequirement = new Requirement('binary');
        this.isWriting = false;
    }

    setValue(event: any): void  {
        if (this.isWriting || (this.newRequirement.value === '' && this.newRequirement.type === 'binary')) {
            this.isWriting = true;
            this.newRequirement.value = event.target.value;
        }
    }

    setName(event: any): void {
        if (this.isWriting || ((this.newRequirement.name === '') && this.newRequirement.type === 'binary')) {
            this.isWriting = true;
            this.newRequirement.name = event.target.value;
        }
    }
}
