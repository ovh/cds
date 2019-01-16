import { Component } from '@angular/core';
import { ModelPattern } from 'app/model/worker-model.model';
import { finalize } from 'rxjs/operators';
import { WorkerModelService } from '../../../../service/worker-model/worker-model.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Table } from '../../../../shared/table/table';

@Component({
    selector: 'app-worker-model-pattern-list',
    templateUrl: './worker-model-pattern.list.html',
    styleUrls: ['./worker-model-pattern.list.scss']
})
export class WorkerModelPatternListComponent extends Table<ModelPattern> {
    workerModelPatterns: Array<ModelPattern> = [];
    filter: string;
    loading = false;
    path: Array<PathItem>;

    constructor(private _workerModelService: WorkerModelService) {
        super();

        this.loading = true;
        this._workerModelService.getWorkerModelPatterns()
            .pipe(finalize(() => this.loading = false))
            .subscribe(wmp => this.workerModelPatterns = wmp);

        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'worker_model_pattern_title',
            routerLink: ['/', 'admin', 'worker-model-pattern']
        }];
    }

    getData(): Array<ModelPattern> {
        if (!this.filter) {
            return this.workerModelPatterns;
        }
        let lowerFilter = this.filter.toLowerCase();

        return this.workerModelPatterns.filter((wmp) => {
            return wmp.name.toLowerCase().indexOf(lowerFilter) !== -1 || wmp.type.toLowerCase() === lowerFilter;
        });
    }
}
