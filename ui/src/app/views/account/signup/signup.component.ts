import {Component} from '@angular/core';
import {Router} from '@angular/router';
import {User} from '../../../model/user.model';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {UserService} from '../../../service/user/user.service';
import {AccountComponent} from '../account.component';

@Component({
    selector: 'app-account-signup',
    templateUrl: './signup.html',
    styleUrls: ['./signup.scss'],
})
export class SignUpComponent extends AccountComponent {

    user: User;
    showWaitingMessage = false;

    constructor(private _userService: UserService, private _router: Router, _authStore: AuthentificationStore) {
        super(_authStore);
        this.user = new User();
    }

    createUser() {
        let bases = document.getElementsByTagName('base');
        let baseHref = null;
        if (bases.length > 0) {
            baseHref = bases[0].href;
        }
        this._userService.signup(this.user, baseHref).subscribe(() => {
            this.showWaitingMessage = true;
        });
    }

    navigateToPassword() {
        this._router.navigate(['/account/password']);
    }

    navigateToLogin() {
        this._router.navigate(['/account/login']);
    }
}
