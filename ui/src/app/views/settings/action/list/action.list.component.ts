import { Component } from '@angular/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { Action } from '../../../../model/action.model';
import { ActionService } from '../../../../service/action/action.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from '../../../../shared/table/data-table.component';

@Component({
    selector: 'app-action-list',
    templateUrl: './action.list.html',
    styleUrls: ['./action.list.scss']
})
export class ActionListComponent {
    loading: boolean;
    columns: Array<Column<Action>>;
    actions: Array<Action>;
    path: Array<PathItem>;
    filter: Filter<Action>;

    constructor(
        private _actionService: ActionService
    ) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                let s = `${d.group.name}/${d.name}`.toLowerCase();
                return s.indexOf(lowerFilter) !== -1;
            }
        };

        this.columns = [
            <Column<Action>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (a: Action) => {
                    return {
                        link: '/settings/action/' + a.group.name + '/' + a.name,
                        value: a.name
                    };
                }
            },
            <Column<Action>>{
                name: 'common_group',
                selector: (a: Action) => a.group.name
            },
            <Column<Action>>{
                type: ColumnType.MARKDOWN,
                name: 'common_description',
                selector: (a: Action) => a.description
            }
        ];
        this.getActions();

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'action_list_title',
            routerLink: ['/', 'settings', 'action']
        }];
    }

    getActions() {
        this.loading = true;
        this._actionService.getAll()
            .pipe(finalize(() => this.loading = false))
            .subscribe(as => { this.actions = as; });
    }
}
