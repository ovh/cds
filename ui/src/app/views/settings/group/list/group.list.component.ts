import { Component } from '@angular/core';
import { finalize } from 'rxjs/operators/finalize';
import { Group } from '../../../../model/group.model';
import { GroupService } from '../../../../service/group/group.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType } from '../../../../shared/table/data-table.component';

@Component({
    selector: 'app-group-list',
    templateUrl: './group.list.html',
    styleUrls: ['./group.list.scss']
})
export class GroupListComponent {
    loading: boolean;
    columns: Array<Column<Group>>;
    groups: Array<Group>;
    path: Array<PathItem>;

    constructor(
        private _groupService: GroupService
    ) {
        this.columns = [
            <Column<Group>>{
                type: ColumnType.ROUTER_LINK,
                name: 'common_name',
                selector: (g: Group) => {
                    return {
                        link: '/settings/group/' + g.name,
                        value: g.name
                    };
                }
            }
        ];
        this.getGroups();

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'group_list_title',
            routerLink: ['/', 'settings', 'group']
        }];
    }

    getGroups(): void {
        this.loading = true;
        this._groupService.getAll()
            .pipe(finalize(() => this.loading = false))
            .subscribe(gs => { this.groups = gs; });
    }

    filter(f: string) {
        const lowerFilter = f.toLowerCase();
        return (g: Group) => {
            return g.name.toLowerCase().indexOf(lowerFilter) !== -1;
        }
    }
}
