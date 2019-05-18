import { Component } from '@angular/core';
import { Broadcast } from 'app/model/broadcast.model';
import { BroadcastStore } from 'app/service/broadcast/broadcast.store';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';

@Component({
    selector: 'app-broadcast-list',
    templateUrl: './broadcast.list.html'
})
export class BroadcastListComponent {
    broadcasts: Array<Broadcast>;
    columns: Array<Column<Broadcast>>;
    path: Array<PathItem>;
    filter: Filter<Broadcast>;

    constructor(
        private _broadcastStore: BroadcastStore
    ) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                return d.id.toString().indexOf(lowerFilter) !== -1 ||
                    d.title.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.level.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.project_key.toLowerCase().indexOf(lowerFilter) !== -1;
            }
        };

        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'broadcast_list_title',
            routerLink: ['/', 'admin', 'broadcast']
        }];

        this.columns = [
            <Column<Broadcast>>{
                type: ColumnType.ROUTER_LINK,
                name: 'broadcast_id',
                class: 'one',
                selector: (b: Broadcast) => {
                    return {
                        link: `/admin/broadcast/${b.id}`,
                        value: b.id
                    };
                }
            },
            <Column<Broadcast>>{
                type: ColumnType.ICON,
                name: 'broadcast_archived',
                class: 'one',
                selector: (b: Broadcast) => { return b.archived ? ['archive', 'icon'] : []; }
            },
            <Column<Broadcast>>{
                type: ColumnType.DATE,
                name: 'broadcast_created',
                class: 'three',
                selector: (b: Broadcast) => b.created
            },
            <Column<Broadcast>>{
                type: ColumnType.ROUTER_LINK,
                name: 'broadcast_title',
                class: 'seven',
                selector: (b: Broadcast) => {
                    return {
                        link: `/admin/broadcast/${b.id}`,
                        value: b.title
                    };
                }
            },
            <Column<Broadcast>>{
                name: 'broadcast_level',
                class: 'two',
                selector: (b: Broadcast) => b.level
            },
            <Column<Broadcast>>{
                name: 'broadcast_project',
                class: 'two',
                selector: (b: Broadcast) => b.project_key
            }
        ];

        this._broadcastStore.getBroadcasts()
            .subscribe(broadcasts => {
                this.broadcasts = broadcasts.valueSeq().toArray()
                    .sort((a, b) => (new Date(b.updated)).getTime() - (new Date(a.updated)).getTime());
            });
    }
}
