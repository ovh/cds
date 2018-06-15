import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Prerequisite} from '../../../model/prerequisite.model';
import {PrerequisiteEvent} from '../prerequisite.event.model';

@Component({
    selector: 'app-prerequisites-form',
    templateUrl: './prerequisites.form.html',
    styleUrls: ['./prerequisites.form.scss']
})
export class PrerequisitesFormComponent {

    newPrerequisite: Prerequisite = new Prerequisite();

    @Input() prerequisites: Array<Prerequisite>;
    @Output() event = new EventEmitter<PrerequisiteEvent>();

    constructor() { }

    addPrerequisite() {
        if (this.newPrerequisite.parameter !== '') {
            this.event.emit(new PrerequisiteEvent('add', this.newPrerequisite));
            this.newPrerequisite = new Prerequisite();
            if (this.prerequisites && this.prerequisites.length > 0) {
                this.newPrerequisite = this.prerequisites[0];
            }
        }
    }
}
