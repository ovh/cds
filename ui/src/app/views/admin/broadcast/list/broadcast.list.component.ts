import { Component, Input } from '@angular/core';
import { Broadcast } from '../../../../model/broadcast.model';
import { BroadcastStore } from '../../../../service/broadcast/broadcast.store';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Table } from '../../../../shared/table/table';

@Component({
    selector: 'app-broadcast-list',
    templateUrl: './broadcast.list.html',
    styleUrls: ['./broadcast.list.scss']
})
export class BroadcastListComponent extends Table<Broadcast> {
    filter: string;
    broadcasts: Array<Broadcast>;

    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };

    path: Array<PathItem>;

    constructor(private _broadcastStore: BroadcastStore) {
        super();

        this._broadcastStore.getBroadcasts().subscribe(broadcasts => {
            this.broadcasts = broadcasts.toArray().sort((a, b) => (new Date(b.updated)).getTime() - (new Date(a.updated)).getTime());
        });

        this.path = [<PathItem>{
            translate: 'common_admin'
        }, <PathItem>{
            translate: 'broadcast_list_title',
            routerLink: ['/', 'admin', 'broadcast']
        }];
    }

    getData(): Array<Broadcast> {
        if (!this.filter) {
            return this.broadcasts;
        }
        let lowerFilter = this.filter.toLowerCase();
        return this.broadcasts.filter(v => v.title.toLowerCase().indexOf(lowerFilter) !== -1);
    }
}
