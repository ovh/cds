import {Component, Input, ViewChild} from '@angular/core';
import {WorkerModel} from '../../../../model/worker-model.model';
import {Table} from '../../../../shared/table/table';
import {WorkerModelService} from '../../../../service/worker-model/worker-model.service';

@Component({
    selector: 'app-worker-model-list',
    templateUrl: './worker-model.list.html',
    styleUrls: ['./worker-model.list.scss']
})
export class WorkerModelListComponent extends Table {
    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };

    public ready = true;
    filter: string;
    workerModels: Array<WorkerModel>;

    constructor(private _workerModelService: WorkerModelService) {
        super();
        this._workerModelService.getWorkerModels().subscribe( wms => {
            this.workerModels = wms;
            this.ready = true;
        });
    }

    getData(): any[] {
        if (!this.filter || this.filter === '') {
            return this.workerModels;
        }
        return this.workerModels.filter(v => v.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
    }
}
