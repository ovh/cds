import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { Broadcast } from 'app/model/broadcast.model';
import { BroadcastStore } from 'app/service/broadcast/broadcast.store';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';

@Component({
    selector: 'app-broadcast-list',
    templateUrl: './broadcast.list.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class BroadcastListComponent {
    broadcasts: Array<Broadcast>;
    columns: Array<Column<Broadcast>>;
    path: Array<PathItem>;
    filter: Filter<Broadcast>;

    constructor(
        private _broadcastStore: BroadcastStore,
        private _cd: ChangeDetectorRef
    ) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => d.id.toString().indexOf(lowerFilter) !== -1 ||
                    d.title.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.level.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.project_key.toLowerCase().indexOf(lowerFilter) !== -1
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
                selector: (b: Broadcast) => ({
                        link: `/admin/broadcast/${b.id}`,
                        value: b.id
                    })
            },
            <Column<Broadcast>>{
                type: ColumnType.DATE,
                name: 'broadcast_created',
                class: 'three',
                selector: (b: Broadcast) => b.created
            },
            <Column<Broadcast>>{
                type: ColumnType.ROUTER_LINK_WITH_ICONS,
                name: 'broadcast_title',
                class: 'eight',
                selector: (b: Broadcast) => {
                    let icons = [];

                    if (b.archived) {
                        icons.push({
                            label: 'broadcast_archived',
                            class: ['archive', 'icon'],
                            title: 'broadcast_archived'
                        });
                    }

                    return {
                        link: `/admin/broadcast/${b.id}`,
                        value: b.title,
                        icons
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
                this._cd.markForCheck();
            });
    }
}
