import {Component, Input, Output, EventEmitter} from '@angular/core';
import {Table} from '../../table/table';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';
import {RequirementStore} from '../../../service/worker/requirement/requirement.store';

@Component({
    selector: 'app-requirements-list',
    templateUrl: './requirements.list.html',
    styleUrls: ['./requirements.list.scss']
})
export class RequirementsListComponent extends Table {

    @Input() requirements: Requirement[];
    @Input() edit: boolean;
    @Output() event = new EventEmitter<RequirementEvent>();

    availableRequirements: Array<string>;


    constructor(private _requirementStore: RequirementStore) {
        super();
        this._requirementStore.getAvailableRequirements().subscribe(r => {
            this.availableRequirements = new Array<string>();
            this.availableRequirements.push(...r.toArray());
        });
    }

    getData(): any[] {
        return this.requirements;
    }

    deleteEvent(r: Requirement): void {
        this.event.emit(new RequirementEvent('delete', r));
    }
}
