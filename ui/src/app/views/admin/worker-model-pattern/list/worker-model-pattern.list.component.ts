import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { ModelPattern } from 'app/model/worker-model.model';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-worker-model-pattern-list',
    templateUrl: './worker-model-pattern.list.html',
    styleUrls: ['./worker-model-pattern.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelPatternListComponent {
    loading: boolean;
    workerModelPatterns: Array<ModelPattern> = [];
    columns: Array<Column<ModelPattern>>;
    path: Array<PathItem>;
    filter: Filter<ModelPattern>;

    constructor(
        private _workerModelService: WorkerModelService,
        private _cd: ChangeDetectorRef
    ) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => d.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.type.toLowerCase().indexOf(lowerFilter) !== -1
        };

        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'worker_model_pattern_title',
            routerLink: ['/', 'admin', 'worker-model-pattern']
        }];

        this.columns = [
            <Column<ModelPattern>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (mp: ModelPattern) => ({
                        link: `/admin/worker-model-pattern/${mp.type}/${mp.name}`,
                        value: mp.name
                    })
            },
            <Column<ModelPattern>>{
                name: 'common_type',
                selector: (mp: ModelPattern) => mp.type
            }
        ];

        this.loading = true;
        this._workerModelService.getPatterns()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wmp => this.workerModelPatterns = wmp);
    }
}
