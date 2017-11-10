import {Component, Input, Output, EventEmitter} from '@angular/core';
import {Table} from '../../table/table';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';
import {RequirementStore} from '../../../service/worker-model/requirement/requirement.store';
import {WorkerModelService} from '../../../service/worker-model/worker-model.service';
import {WorkerModel} from '../../../model/worker-model.model';

@Component({
    selector: 'app-requirements-list',
    templateUrl: './requirements.list.html',
    styleUrls: ['./requirements.list.scss']
})
export class RequirementsListComponent extends Table {

    @Input() requirements: Requirement[];
    @Input() edit: boolean;
    @Input('suggest')
    set suggest(data: string[]) {
        if (data) {
            this._suggest = data;
        } else {
            this._suggest = [];
        }
    }
    get suggest() {
        return this._suggest;
    }
    @Output() event = new EventEmitter<RequirementEvent>();
    @Output() onChange = new EventEmitter<Requirement[]>();

    availableRequirements: Array<string>;
    workerModels: Array<string>;
    _suggest: string[] = [];
    loading = true;

    constructor(private _requirementStore: RequirementStore, private _workerModelService: WorkerModelService) {
        super();
        this._requirementStore.getAvailableRequirements()
            .subscribe(r => {
                this.availableRequirements = new Array<string>();
                this.availableRequirements.push(...r.toArray());
            });

        this._workerModelService.getWorkerModels()
        .first()
        .finally(() => this.loading = false)
        .subscribe(wms => {
            this.workerModels = wms.map((wm) => wm.name).concat(this.workerModels);
        });
    }

    getData(): any[] {
        return this.requirements;
    }

    deleteEvent(r: Requirement): void {
        this.event.emit(new RequirementEvent('delete', r));
    }

    change(): void {
        this.onChange.emit(this.requirements);
    }
}
