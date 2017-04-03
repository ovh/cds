import {Component} from '@angular/core';
import {User} from '../../../model/user.model';
import {UserService} from '../../../service/user/user.service';
import {Router, ActivatedRoute} from '@angular/router';
import {AccountComponent} from '../account.component';
import {AuthentificationStore} from '../../../service/auth/authentification.store';

@Component({
    selector: 'app-account-signup',
    templateUrl: './signup.html',
    styleUrls: ['./signup.scss'],
})
export class SignUpComponent extends AccountComponent {

    user: User;
    showWaitingMessage = false;

    constructor(private _userService: UserService, private _router: Router,
                private t: ActivatedRoute, private _authStore: AuthentificationStore) {
        super(_authStore);
        this.user = new User();
    }

    createUser() {
        this._userService.signup(this.user, location.origin).subscribe(res => {
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
