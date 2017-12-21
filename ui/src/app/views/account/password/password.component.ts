import {Component} from '@angular/core';
import {User} from '../../../model/user.model';
import {UserService} from '../../../service/user/user.service';
import {Router} from '@angular/router';
import {AccountComponent} from '../account.component';
import {AuthentificationStore} from '../../../service/auth/authentification.store';

@Component({
    selector: 'app-account-password',
    templateUrl: './password.html',
    styleUrls: ['./password.scss'],
})
export class PasswordComponent extends AccountComponent {

    user: User;
    showWaitingMessage = false;

    constructor(private _userService: UserService, private _router: Router, _authStore: AuthentificationStore) {
        super(_authStore);
        this.user = new User();
    }

    resetPassword() {
        let bases = document.getElementsByTagName('base');
        let baseHref = null;
        if (bases.length > 0) {
            baseHref = bases[0].href;
        }
        this._userService.resetPassword(this.user, baseHref).subscribe(() => {
            this.showWaitingMessage = true;
        });
    }

    navigateToSignUp() {
        this._router.navigate(['/account/signup']);
    }

    navigateToLogin() {
        this._router.navigate(['/account/login']);
    }
}
