import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { finalize } from 'rxjs/operators';
import { Action } from '../../../../model/action.model';
import { ActionService } from '../../../../service/action/action.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';
import { Tab } from '../../../../shared/tabs/tabs.component';

@Component({
    selector: 'app-action-list',
    templateUrl: './action.list.html',
    styleUrls: ['./action.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ActionListComponent {
    loading: boolean;
    loadingBuiltin: boolean;
    columns: Array<Column<Action>>;
    columnsBuiltin: Array<Column<Action>>;
    actions: Array<Action>;
    actionsBuiltin: Array<Action>;
    path: Array<PathItem>;
    tabs: Array<Tab>;
    selectedTab: Tab;

    constructor(
        private _actionService: ActionService,
        private _cd: ChangeDetectorRef
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

        this.columns = [
            <Column<Action>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (a: Action) => ({
                        link: `/settings/action/${a.group.name}/${a.name}`,
                        value: a.name
                    })
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
                selector: (a: Action) => ({
                        link: `/settings/action-builtin/${a.name}`,
                        value: a.name
                    })
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

    getActions() {
        this.loading = true;
        this._actionService.getAll()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(as => {
 this.actions = as;
});
    }

    getActionBuiltins() {
        this.loadingBuiltin = true;
        this._actionService.getAllBuiltin()
            .pipe(finalize(() => {
                this.loadingBuiltin = false;
                this._cd.markForCheck();
            }))
            .subscribe(as => {
 this.actionsBuiltin = as;
});
    }

    filter(f: string) {
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
                this.getActions();
                break;
            case 'builtin':
                this.getActionBuiltins();
                break;
        }
        this.selectedTab = tab;
    }
}
