import {Component, Output, EventEmitter} from '@angular/core';
import {RequirementStore} from '../../../service/worker/requirement/requirement.store';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';

@Component({
    selector: 'app-requirements-form',
    templateUrl: './requirements.form.html',
    styleUrls: ['./requirements.form.scss']
})
export class RequirementsFormComponent {

    @Output() event = new EventEmitter<RequirementEvent>();

    newRequirement: Requirement = new Requirement('binary');
    availableRequirements: Array<string>;
    isWriting = false;

    constructor(private _requirementStore: RequirementStore) {
        this._requirementStore.getAvailableRequirements().subscribe(r => {
            this.availableRequirements = new Array<string>();
            this.availableRequirements.push(...r.toArray());
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
