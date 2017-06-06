import {Component, OnInit} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {Action} from '../../../../model/action.model';
import {ActionService} from '../../../../service/action/action.service';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';

@Component({
    selector: 'app-action-edit',
    templateUrl: './action.edit.html',
    styleUrls: ['./action.edit.scss']
})
export class ActionEditComponent implements OnInit {
    action: Action;
    isAdmin: boolean;

    private namePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    private actionPatternError = false;

    constructor(private _actionService: ActionService,
                private _route: ActivatedRoute,
                private _authentificationStore: AuthentificationStore) {
        if (this._authentificationStore.isConnected()) {
            this.isAdmin = this._authentificationStore.isAdmin();
        }
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            if (params['name'] !== 'add') {
                this._actionService.getAction(params['name']).subscribe( u => {
                    this.action = u;
                });
            } else {
                this.action = new Action();
            }
        });
    }

}
