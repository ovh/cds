import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { SharedInfraGroupName } from 'app/model/group.model';
import { Requirement } from 'app/model/requirement.model';
import { WorkerModel } from 'app/model/worker-model.model';
import { RequirementStore } from 'app/service/requirement/requirement.store';
import { RequirementEvent } from 'app/shared/requirements/requirement.event.model';
import { Table } from 'app/shared/table/table';
import { first } from 'rxjs/operators';

export const OSArchitecture = 'os-architecture';

@Component({
    selector: 'app-requirements-list',
    templateUrl: './requirements.list.html',
    styleUrls: ['./requirements.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RequirementsListComponent extends Table<Requirement> implements OnInit {
    @Input() requirements: Requirement[];
    @Input() edit: boolean;

    _suggest: string[] = [];
    @Input() set suggest(data: string[]) {
        if (data) {
            this._suggest = data;
        } else {
            this._suggest = [];
        }
    }
    get suggest() { return this._suggest; }

    _workerModels: Array<WorkerModel>;
    @Input() set workerModels(wms: Array<WorkerModel>) {
        if (wms) {
            this._workerModels = wms;

            this.suggestWithWorkerModel = wms.map(wm => {
                if (wm.group.name !== SharedInfraGroupName) {
                    return `${wm.group.name}/${wm.name}`;
                }
                return wm.name;
            }).concat(this._suggest);
        }
    }
    get workerModels() { return this._workerModels; }

    @Output() event = new EventEmitter<RequirementEvent>();
    @Output() onChange = new EventEmitter<Requirement[]>();

    availableRequirements: Array<string>;
    suggestWithWorkerModel: Array<string> = [];
    suggestWithOSArch: Array<string> = [];

    constructor(
        private _requirementStore: RequirementStore,
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
        this._requirementStore.getRequirementsTypeValues('os-architecture').pipe(first()).subscribe(values => {
            this.suggestWithOSArch = values.concat(this.suggest);
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
