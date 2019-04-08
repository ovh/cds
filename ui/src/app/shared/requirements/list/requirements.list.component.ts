import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { SharedInfraGroupName } from 'app/model/group.model';
import { finalize, first } from 'rxjs/operators';
import { Requirement } from '../../../model/requirement.model';
import { WorkerModel } from '../../../model/worker-model.model';
import { RequirementStore } from '../../../service/requirement/requirement.store';
import { WorkerModelService } from '../../../service/worker-model/worker-model.service';
import { Table } from '../../table/table';
import { RequirementEvent } from '../requirement.event.model';

export const OSArchitecture = 'os-architecture';

@Component({
    selector: 'app-requirements-list',
    templateUrl: './requirements.list.html',
    styleUrls: ['./requirements.list.scss']
})
export class RequirementsListComponent extends Table<Requirement> implements OnInit {
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

    get suggestWithWorkerModel() {
        return this._suggestWithWorkerModel;
    }

    get suggestWithOSArch() {
        return this._suggestWithOSArch;
    }

    @Output() event = new EventEmitter<RequirementEvent>();
    @Output() onChange = new EventEmitter<Requirement[]>();

    availableRequirements: Array<string>;
    workerModels: Array<WorkerModel>;
    _suggest: string[] = [];
    _suggestWithWorkerModel: Array<string> = [];
    _suggestWithOSArch: Array<string> = [];

    loading = true;

    constructor(
        private _requirementStore: RequirementStore,
        private _workerModelService: WorkerModelService,
    ) {
        super();
        this.nbElementsByPage = 5;

        this._requirementStore.getAvailableRequirements()
            .subscribe(r => {
                this.availableRequirements = new Array<string>();
                this.availableRequirements.push(...r.toArray());
            });
    }

    ngOnInit() {
        this._workerModelService.getAll(null)
            .pipe(finalize(() => this.loading = false), first())
            .subscribe(wms => {
                this.workerModels = wms;
                if (Array.isArray(this.workerModels)) {
                    this._suggestWithWorkerModel = this.workerModels.map(wm => {
                        if (wm.group.name !== SharedInfraGroupName) {
                            return `${wm.group.name}/${wm.name}`;
                        }
                        return wm.name;
                    }).concat(this._suggest);
                }
            });

        this._requirementStore.getRequirementsTypeValues('os-architecture').pipe(first()).subscribe(values => {
            this._suggestWithOSArch = values.concat(this.suggest);
        });
    }

    getData(): Array<Requirement> {
        return this.requirements;
    }

    deleteEvent(r: Requirement): void {
        this.event.emit(new RequirementEvent('delete', r));
    }

    change(req: Requirement): void {
        switch (req.type) {
            case 'service':
                // if type service, user have to choose a hostname
                break;
            case 'memory':
                // memory: memory_4096
                req.name = 'memory_' + req.value;
                break
            case 'model':
                req.name = req.value;
                break
            case 'volume':
                break;
            case OSArchitecture:
                req.name = OSArchitecture;
                break;
            default:
                // else, name is the value of the requirement
                req.name = req.value;
        }
        this.onChange.emit(this.requirements);
    }

    getWorkerModel(name: string): WorkerModel {
        return this.workerModels.find(m => m.name === name);
    }
}
