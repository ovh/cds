import {Component} from '@angular/core';
import {WorkerModel} from '../../../../model/worker-model.model';
import {Table} from '../../../../shared/table/table';
import {WorkerModelService} from '../../../../service/worker-model/worker-model.service';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-worker-model-list',
    templateUrl: './worker-model.list.html',
    styleUrls: ['./worker-model.list.scss']
})
export class WorkerModelListComponent extends Table {
    filter: string;
    binaryValue: string;
    loading = true;
    searchView = true;
    workerModels: Array<WorkerModel>;
    filteredWorkerModels: Array<WorkerModel>;

    constructor(private _workerModelService: WorkerModelService) {
        super();
        this._workerModelService.getWorkerModels()
            .pipe(finalize(() => this.loading = false))
            .subscribe(wms => {
                this.workerModels = wms;
                this.filteredWorkerModels = wms;
            });
        this.nbElementsByPage = 25;

    }

    getData(): any[] {
        if (!this.filter) {
            return this.filteredWorkerModels;
        }
        let lowerFilter = this.filter.toLowerCase();

        return this.filteredWorkerModels.filter((v) => {
          return v.name.toLowerCase().indexOf(lowerFilter) !== -1 || v.type.toLowerCase() === lowerFilter;
        });
    }

    getImageName(w: WorkerModel): string {
        if (w.type === 'docker') {
            if (w.model_docker != null && w.model_docker.image) {
                return w.model_docker.image.substr(0, 60)
            }
        } else {
            if (w.model_virtual_machine != null && w.model_virtual_machine.image) {
                return w.model_virtual_machine.image.substr(0, 60)
            }
        }
        return '';
    }

    searchBinary(binary: string) {
        this.filter = '';
        if (!binary) {
            this.searchView = true;
            this.filteredWorkerModels = this.workerModels;
            this.binaryValue = '';
            return;
        }
        this._workerModelService.getWorkerModels(binary)
            .pipe(finalize(() => {
                this.loading = false;
                this.searchView = false;
            }))
            .subscribe((wms) => this.filteredWorkerModels = wms);
    }
}
