import {Component} from '@angular/core';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {User} from '../../../model/user.model';

@Component({
    selector: 'app-settings-sidebar',
    templateUrl: './settings-sidebar.html',
    styleUrls: ['./settings-sidebar.scss']
})
export class SettingsSidebarComponent {
    currentUser: User;
    constructor(private _authentificationStore: AuthentificationStore) {
      this.currentUser = this._authentificationStore.getUser();
    }
}
