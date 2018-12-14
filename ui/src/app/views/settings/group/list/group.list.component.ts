import { Component, Input } from '@angular/core';
import { Group } from '../../../../model/group.model';
import { GroupService } from '../../../../service/group/group.service';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Table } from '../../../../shared/table/table';

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
    path: Array<PathItem>;

    constructor(private _groupService: GroupService) {
        super();

        this._groupService.getGroups().subscribe(wms => {
            this.groups = wms;
        });

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'group_list_title',
            routerLink: ['/', 'settings', 'group']
        }];
    }

    getData(): any[] {
        if (!this.filter) {
            return this.groups;
        }
        return this.groups.filter(v => v.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
    }
}
