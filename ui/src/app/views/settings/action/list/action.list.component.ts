import { Component, Input } from '@angular/core';
import { Action } from '../../../../model/action.model';
import { User } from '../../../../model/user.model';
import { ActionService } from '../../../../service/action/action.service';
import { AuthentificationStore } from '../../../../service/auth/authentification.store';
import { PathItem } from '../../../../shared/breadcrumb/breadcrumb.component';
import { Table } from '../../../../shared/table/table';

@Component({
    selector: 'app-action-list',
    templateUrl: './action.list.html',
    styleUrls: ['./action.list.scss']
})
export class ActionListComponent extends Table {
    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };

    filter: string;
    actions: Array<Action>;
    currentUser: User;
    path: Array<PathItem>;

    constructor(
        private _actionService: ActionService,
        private _authentificationStore: AuthentificationStore
    ) {
        super();

        this.currentUser = this._authentificationStore.getUser();
        this._actionService.getActions().subscribe(actions => {
            this.actions = actions;
        });

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'action_list_title',
            routerLink: ['/', 'settings', 'action']
        }];
    }

    getData(): any[] {
        if (!this.filter) {
            return this.actions;
        }
        return this.actions.filter(v => v.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
    }
}
