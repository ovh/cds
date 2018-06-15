import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Prerequisite} from '../../../model/prerequisite.model';
import {Table} from '../../table/table';
import {PrerequisiteEvent} from '../prerequisite.event.model';

@Component({
    selector: 'app-prerequisites-list',
    templateUrl: './prerequisites.html',
    styleUrls: ['./prerequisites.scss']
})
export class PrerequisiteComponent extends Table {

    @Input() prerequisites: Prerequisite[];
    @Input() edit = false;
    @Output() event = new EventEmitter<PrerequisiteEvent>();

    constructor() {
        super();
        this.nbElementsByPage = 5;
    }

    getData(): any[] {
        return this.prerequisites;
    }

    remove(p): void {
        this.event.emit(new PrerequisiteEvent('delete', p));
    }

    update(p): void {
        this.event.emit(new PrerequisiteEvent('update', p));
    }
}
