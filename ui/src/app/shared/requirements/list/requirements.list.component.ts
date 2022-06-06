import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output
} from '@angular/core';
import { SharedInfraGroupName } from 'app/model/group.model';
import { Requirement } from 'app/model/requirement.model';
import { WorkerModel } from 'app/model/worker-model.model';
import { RequirementStore } from 'app/service/requirement/requirement.store';
import { RequirementEvent } from 'app/shared/requirements/requirement.event.model';
import { first } from 'rxjs/operators';

export const OSArchitecture = 'os-architecture';

@Component({
    selector: 'app-requirements-list',
    templateUrl: './requirements.list.html',
    styleUrls: ['./requirements.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RequirementsListComponent implements OnInit {
    @Input() requirements: Requirement[];
    @Input() edit: boolean;

    @Input() suggest: Array<string> = [];

    @Input() set workerModels(wms: Array<WorkerModel>) {
        if (wms) {
            this.workerModelsMap = new Map<string, WorkerModel>();
            this.suggestWithWorkerModel = new Array<string>();
            if (wms) {
                wms.forEach(wm => {
                    let name = wm.name;
                    if (wm.group.name !== SharedInfraGroupName) {
                        name = `${wm.group.name}/${wm.name}`;
                    }
                    this.suggestWithWorkerModel.push(name);
                    this.workerModelsMap.set(name, wm);
                })
            }
        }
    }
    workerModelsMap: Map<string, WorkerModel> = new Map();

    @Output() event = new EventEmitter<RequirementEvent>();
    @Output() onChange = new EventEmitter<Requirement[]>();

    availableRequirements: Array<string>;
    suggestWithWorkerModel: Array<string> = [];
    suggestWithOSArch: Array<string> = [];

    constructor(
        private _requirementStore: RequirementStore,
        private _cd: ChangeDetectorRef
    ) {
        this._requirementStore.getAvailableRequirements()
            .subscribe(r => {
                this.availableRequirements = new Array<string>();
                this.availableRequirements.push(...r.toArray());
            });
    }

    ngOnInit() {
        this._requirementStore.getRequirementsTypeValues('os-architecture').pipe(first()).subscribe(values => {
            this.suggestWithOSArch = values.concat(this.suggest);
            this._cd.markForCheck();
        });
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
                break;
            case 'model':
                req.name = req.value;
                break;
            case OSArchitecture:
                req.name = OSArchitecture;
                break;
            default:
                // else, name is the value of the requirement
                req.name = req.value;
        }

        this.onChange.emit(this.requirements);
        this._cd.markForCheck();
    }
}
