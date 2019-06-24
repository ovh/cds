import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Prerequisite } from 'app/model/prerequisite.model';
import { PrerequisiteEvent } from 'app/shared/prerequisites/prerequisite.event.model';
import { Table } from 'app/shared/table/table';

@Component({
    selector: 'app-prerequisites-list',
    templateUrl: './prerequisites.html',
    styleUrls: ['./prerequisites.scss']
})
export class PrerequisiteComponent extends Table<Prerequisite> {

    @Input() prerequisites: Prerequisite[];
    @Input() edit = false;
    @Output() event = new EventEmitter<PrerequisiteEvent>();

    constructor() {
        super();
        this.nbElementsByPage = 5;
    }

    getData(): Array<Prerequisite> {
        return this.prerequisites;
    }

    remove(p): void {
        this.event.emit(new PrerequisiteEvent('delete', p));
    }

    update(p): void {
        this.event.emit(new PrerequisiteEvent('update', p));
    }
}
