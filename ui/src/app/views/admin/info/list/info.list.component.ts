import {Component, Input} from '@angular/core';
import {Info} from '../../../../model/info.model';
import {Table} from '../../../../shared/table/table';
import {InfoService} from '../../../../service/info/info.service';

@Component({
    selector: 'app-info-list',
    templateUrl: './info.list.html',
    styleUrls: ['./info.list.scss']
})
export class InfoListComponent extends Table {
    filter: string;
    infos: Array<Info>;

    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };

    constructor(private _infoService: InfoService) {
        super();
        this._infoService.getInfos().subscribe( infos => {
            this.infos = infos;
        });
    }

    getData(): any[] {
        if (!this.filter) {
            return this.infos;
        }
        return this.infos.filter(v => v.title.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
    }
}
