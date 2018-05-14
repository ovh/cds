import {Component} from '@angular/core';
import {WorkerModel} from '../../../../model/worker-model.model';
import {Table} from '../../../../shared/table/table';
import {WorkerModelService} from '../../../../service/worker-model/worker-model.service';

@Component({
    selector: 'app-worker-model-list',
    templateUrl: './worker-model.list.html',
    styleUrls: ['./worker-model.list.scss']
})
export class WorkerModelListComponent extends Table {
    filter: string;
    workerModels: Array<WorkerModel>;

    constructor(private _workerModelService: WorkerModelService) {
        super();
        this._workerModelService.getWorkerModels().subscribe( wms => {
            this.workerModels = wms;
        });
        this.nbElementsByPage = 25;

    }

    getData(): any[] {
        if (!this.filter) {
            return this.workerModels;
        }
        let lowerFilter = this.filter.toLowerCase();

        return this.workerModels.filter((v) => {
          return v.name.toLowerCase().indexOf(lowerFilter) !== -1 || v.type.toLowerCase() === lowerFilter;
        });
    }

    getImageName(w: WorkerModel): string {
        if (w.type === 'docker') {
            if (w.model_docker != null && w.model_docker.image) {
                return w.model_docker.image.substr(0, 60)
            } else {
                console.log('error model docker : ', w);
            }
        } else {
            if (w.model_virtual_machine != null && w.model_virtual_machine.image) {
                return w.model_virtual_machine.image.substr(0, 60)
            } else {
                console.log('error model virtual : ', w);
            }
        }
        return '';
    }
}
