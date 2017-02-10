import {Component} from '@angular/core';
import {User} from '../../../model/user.model';
import {UserService} from '../../../service/user/user.service';
import {Router} from '@angular/router';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {AccountComponent} from '../account.component';

@Component({
    selector: 'app-account-login',
    templateUrl: './login.html',
    styleUrls: ['./login.scss']
})
export class LoginComponent extends AccountComponent {

    user: User;

    constructor(private _userService: UserService, private _router: Router, private _authStore: AuthentificationStore) {
        super(_authStore);
        this.user = new User();
    }

    signIn() {
        this._userService.login(this.user).subscribe(() => {
            this._router.navigate(['home']);
        });
    }

    navigateToSignUp() {
        this._router.navigate(['/account/signup']);
    }

    navigateToPassword() {
        this._router.navigate(['/account/password']);
    }
}
