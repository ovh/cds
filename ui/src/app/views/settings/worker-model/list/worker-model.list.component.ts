import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { WorkerModel } from 'app/model/worker-model.model';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-worker-model-list',
    templateUrl: './worker-model.list.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelListComponent {
    loading: boolean;
    columns: Array<Column<WorkerModel>>;
    workerModels: Array<WorkerModel>;
    path: Array<PathItem>;
    filter: Filter<WorkerModel>;
    selectedState: string;
    binaryValue: string;
    binarySelected: boolean;

    constructor(
        private _workerModelService: WorkerModelService,
        private _cd: ChangeDetectorRef
    ) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                let s = `${d.group.name}/${d.name}`.toLowerCase();
                return s.indexOf(lowerFilter) !== -1 ||
                    d.type.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.description.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    this.getImageName(d).toLowerCase().indexOf(lowerFilter) !== -1;
            }
        };

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'worker_model_list_title',
            routerLink: ['/', 'settings', 'worker-model']
        }];

        this.columns = [
            <Column<WorkerModel>>{
                name: 'worker_model_group',
                class: 'three',
                selector: (wm: WorkerModel) => wm.group.name
            },
            <Column<WorkerModel>>{
                type: ColumnType.ROUTER_LINK_WITH_ICONS,
                name: 'common_name',
                class: 'four',
                selector: (wm: WorkerModel) => {
                    let icons = [];

                    if (wm.disabled) {
                        icons.push({
                            label: 'worker_model_disabled',
                            class: ['grey', 'ban', 'icon'],
                            title: 'worker_model_disabled'
                        });
                    }
                    if (wm.nb_spawn_err > 0) {
                        icons.push({
                            label: 'worker_model_spawn_error_tooltip',
                            class: ['exclamation', 'triangle', 'icon', 'red'],
                            title: 'worker_model_warning'
                        });
                    }
                    if (wm.is_official) {
                        icons.push({
                            label: 'worker_model_official_tooltip',
                            class: ['check', 'circle', 'outline', 'icon', 'green'],
                            title: 'worker_model_official_tooltip'
                        });
                    }
                    if (wm.is_deprecated) {
                        icons.push({
                            label: 'worker_model_deprecated_tooltip',
                            class: ['exclamation', 'circle', 'icon', 'orange'],
                            title: 'worker_model_official_tooltip'
                        });
                    }

                    return {
                        link: `/settings/worker-model/${wm.group.name}/${wm.name}`,
                        value: wm.name,
                        icons
                    };
                }
            },
            <Column<WorkerModel>>{
                name: 'common_description',
                class: 'four',
                selector: (wm: WorkerModel) => wm.description
            },
            <Column<WorkerModel>>{
                name: 'common_type',
                class: 'two',
                selector: (wm: WorkerModel) => wm.type
            },
            <Column<WorkerModel>>{
                name: 'worker_model_image',
                class: 'three',
                selector: this.getImageName
            },
        ];

        this.loadWorkerModels();
    }

    loadWorkerModels() {
        this.loading = true;
        this._workerModelService.getAll(this.selectedState, this.binaryValue)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wms => {
                this.workerModels = wms.sort((a, b) => {
                    let aG = a.group.name.toLowerCase();
                    let bG = b.group.name.toLowerCase();
                    if (aG === bG) {
                        return a.name.toLowerCase() > b.name.toLowerCase() ? 1 : -1;
                    }
                    return aG > bG ? 1 : -1;
                });
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

    selectState() {
        this.loadWorkerModels();
    }

    resetBinary() {
        this.binarySelected = false;
        this.binaryValue = null;
        this.loadWorkerModels();
    }

    searchBinary() {
        this.binarySelected = true;
        this.loadWorkerModels();
    }
}
