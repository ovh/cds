import {Component} from '@angular/core';
import {finalize} from 'rxjs/operators';
import {WorkerModel} from '../../../../model/worker-model.model';
import {WorkerModelService} from '../../../../service/worker-model/worker-model.service';
import {Table} from '../../../../shared/table/table';

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
    ready = false;
    set selectedFilter(filter: string) {
        this._selectedFilter = filter;
        if (this.ready) {
            this.loadWorkerModels(this._selectedFilter);
        }
    }
    get selectedFilter(): string {
        return this._selectedFilter;
    }

    _selectedFilter: string;

    constructor(private _workerModelService: WorkerModelService) {
        super();
        this.loadWorkerModels(null);
        this.nbElementsByPage = 25;
    }

    loadWorkerModels(filter: string) {
        this.binaryValue = '';
        this.loading = true;
        this._workerModelService.getWorkerModels(filter)
            .pipe(finalize(() => this.loading = false))
            .subscribe(wms => {
                this.workerModels = wms;
                this.filteredWorkerModels = wms;
                this.ready = true;
            });
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
        this.selectedFilter = null;
        if (!binary) {
            this.searchView = true;
            this.filteredWorkerModels = this.workerModels;
            this.binaryValue = '';
            this.currentPage = 1;
            return;
        }
        this._workerModelService.getWorkerModels(this.selectedFilter, binary)
            .pipe(finalize(() => {
                this.loading = false;
                this.searchView = false;
            }))
            .subscribe((wms) => this.filteredWorkerModels = wms);
    }
}
