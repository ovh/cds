import {Component, Input} from '@angular/core';
import {Group} from '../../../../model/group.model';
import {Table} from '../../../../shared/table/table';
import {GroupService} from '../../../../service/group/group.service';

@Component({
    selector: 'app-group-list',
    templateUrl: './group.list.html',
    styleUrls: ['./group.list.scss']
})
export class GroupListComponent extends Table {
    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };

    filter: string;
    groups: Array<Group>;

    constructor(private _groupService: GroupService) {
        super();
        this._groupService.getGroups().subscribe( wms => {
            this.groups = wms;
        });
    }

    getData(): any[] {
        if (!this.filter) {
            return this.groups;
        }
        return this.groups.filter(v => v.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
    }
}
