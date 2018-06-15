import {Component, Input} from '@angular/core';
import {Broadcast} from '../../../../model/broadcast.model';
import {BroadcastService} from '../../../../service/broadcast/broadcast.service';
import {Table} from '../../../../shared/table/table';

@Component({
    selector: 'app-broadcast-list',
    templateUrl: './broadcast.list.html',
    styleUrls: ['./broadcast.list.scss']
})
export class BroadcastListComponent extends Table {
    filter: string;
    broadcasts: Array<Broadcast>;

    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };

    constructor(private _broadcastService: BroadcastService) {
        super();
        this._broadcastService.getBroadcasts().subscribe( broadcasts => {
            this.broadcasts = broadcasts;
        });
    }

    getData(): any[] {
        if (!this.filter) {
            return this.broadcasts;
        }
        let lowerFilter = this.filter.toLowerCase();
        return this.broadcasts.filter(v => v.title.toLowerCase().indexOf(lowerFilter) !== -1);
    }
}
