import {Component} from '@angular/core';
import { ModelPattern } from 'app/model/worker-model.model';
import {finalize} from 'rxjs/operators';
import {WorkerModelService} from '../../../service/worker-model/worker-model.service';
import {Table} from '../../../shared/table/table';

@Component({
    selector: 'app-worker-model-pattern',
    templateUrl: './worker-model-pattern.html',
    styleUrls: ['./worker-model-pattern.scss']
})
export class WorkerModelPatternComponent extends Table {

    workerModelPatterns: Array<ModelPattern> = [];
    filter: string;
    loading = false;

    constructor(private _workerModelService: WorkerModelService) {
        super();
        this.loading = true;
        this._workerModelService.getWorkerModelPatterns()
            .pipe(finalize(() => this.loading = false))
            .subscribe(wmp => this.workerModelPatterns = wmp);
    }

    getData(): any[] {
        if (!this.filter) {
            return this.workerModelPatterns;
        }
        let lowerFilter = this.filter.toLowerCase();

        return this.workerModelPatterns.filter((wmp) => {
          return wmp.name.toLowerCase().indexOf(lowerFilter) !== -1 || wmp.type.toLowerCase() === lowerFilter;
        });
    }
}
