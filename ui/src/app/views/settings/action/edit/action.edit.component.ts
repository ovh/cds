import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Action} from '../../../../model/action.model';
import {ActionEvent} from '../../../../shared/action/action.event.model';
import {ActionService} from '../../../../service/action/action.service';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';

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
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore) {
        if (this._authentificationStore.isConnected()) {
            this.isAdmin = this._authentificationStore.isAdmin();
        }
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            this._actionService.getAction(params['name']).subscribe( u => {
                this.action = u;
            });
        });
    }

    actionEvent(event: ActionEvent): void {
        event.action.loading = true;
        switch (event.type) {
            case 'update':
                this._actionService.updateAction(this.action.name, event.action).subscribe( action => {
                    this._toast.success('', this._translate.instant('action_saved'));
                    this.action = action;
                });
                break;
            case 'delete':
                this._actionService.deleteAction(event.action.name).subscribe( () => {
                    this._toast.success('', this._translate.instant('action_deleted'));
                    this._router.navigate(['settings', 'action']);
                });
                break;
        }
    }

}
