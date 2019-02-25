import { Component } from '@angular/core';
import { finalize } from 'rxjs/internal/operators/finalize';
import { Action } from '../../../../model/action.model';
import { ActionService } from '../../../../service/action/action.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';
import { Tab } from '../../../../shared/tabs/tabs.component';

@Component({
    selector: 'app-action-list',
    templateUrl: './action.list.html',
    styleUrls: ['./action.list.scss']
})
export class ActionListComponent {
    loadingCustom: boolean;
    loadingBuiltin: boolean;
    columnsCustom: Array<Column<Action>>;
    columnsBuiltin: Array<Column<Action>>;
    actionsCustom: Array<Action>;
    actionsBuiltin: Array<Action>;
    path: Array<PathItem>;
    tabs: Array<Tab>;
    selectedTab: Tab;

    constructor(
        private _actionService: ActionService
    ) {
        this.tabs = [<Tab>{
            translate: 'action_custom',
            icon: '',
            key: 'custom',
            default: true
        }, <Tab>{
            translate: 'action_builtin',
            icon: '',
            key: 'builtin'
        }];

        this.columnsCustom = [
            <Column<Action>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (a: Action) => {
                    return {
                        link: `/settings/action/custom/${a.group.name}/${a.name}`,
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

        this.columnsBuiltin = [
            <Column<Action>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (a: Action) => {
                    return {
                        link: `/settings/action/builtin/${a.name}`,
                        value: a.name
                    };
                }
            },
            <Column<Action>>{
                type: ColumnType.MARKDOWN,
                name: 'common_description',
                selector: (a: Action) => a.description
            }
        ];

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'action_list_title',
            routerLink: ['/', 'settings', 'action']
        }];
    }

    getCustomActions() {
        this.loadingCustom = true;
        this._actionService.getAll()
            .pipe(finalize(() => this.loadingCustom = false))
            .subscribe(as => { this.actionsCustom = as; });
    }

    getBuiltinActions() {
        this.loadingBuiltin = true;
        this._actionService.getAll()
            .pipe(finalize(() => this.loadingBuiltin = false))
            .subscribe(as => { this.actionsBuiltin = as; });
    }

    filterCustom(f: string) {
        const lowerFilter = f.toLowerCase();
        return (d: Action) => {
            let s = `${d.group.name}/${d.name}`.toLowerCase();
            return s.indexOf(lowerFilter) !== -1;
        }
    }

    filterBuiltin(f: string) {
        const lowerFilter = f.toLowerCase();
        return (d: Action) => {
            let s = d.name.toLowerCase();
            return s.indexOf(lowerFilter) !== -1;
        }
    }

    selectTab(tab: Tab): void {
        switch (tab.key) {
            case 'custom':
                this.getCustomActions();
                break;
            case 'builtin':
                this.getBuiltinActions();
                break;
        }
        this.selectedTab = tab;
    }
}
