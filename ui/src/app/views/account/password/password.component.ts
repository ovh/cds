import {Component} from '@angular/core';
import {User} from '../../../model/user.model';
import {UserService} from '../../../service/user/user.service';
import {Router, ActivatedRoute} from '@angular/router';
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

    constructor(private _userService: UserService, private _router: Router,
                private t: ActivatedRoute, private _authStore: AuthentificationStore) {
        super(_authStore);
        this.user = new User();
    }

    resetPassword() {
        this._userService.resetPassword(this.user, location.origin).subscribe(res => {
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
