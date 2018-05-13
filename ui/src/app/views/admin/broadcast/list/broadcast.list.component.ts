import {Component, Input} from '@angular/core';
import {Broadcast} from '../../../../model/broadcast.model';
import {Table} from '../../../../shared/table/table';
import {BroadcastService} from '../../../../service/broadcast/broadcast.service';

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
        return this.broadcasts.filter(v => v.title.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
    }
}
