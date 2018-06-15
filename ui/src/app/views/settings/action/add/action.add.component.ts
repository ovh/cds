import {Component} from '@angular/core';
import {Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {Action} from '../../../../model/action.model';
import {ActionService} from '../../../../service/action/action.service';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {ActionEvent} from '../../../../shared/action/action.event.model';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-action-add',
    templateUrl: './action.add.html',
    styleUrls: ['./action.add.scss']
})
export class ActionAddComponent {
    action: Action;
    isAdmin: boolean;

    constructor(private _actionService: ActionService,
                private _toast: ToastService, private _translate: TranslateService,
                private _router: Router,
                private _authentificationStore: AuthentificationStore) {
        this.action = new Action();
        this.action.enabled = true;
        if (this._authentificationStore.isConnected()) {
            this.isAdmin = this._authentificationStore.isAdmin();
        }
    }

    actionEvent(event: ActionEvent): void {
        this.action.loading = true;
        this._actionService.createAction(event.action).subscribe( action => {
            this._toast.success('', this._translate.instant('action_saved'));
            // navigate to have action name in url
            this._router.navigate(['settings', 'action', event.action.name]);
        }, () => {
            this.action.loading = false;
        });
    }

    // TODO check name pattern before submit

}
