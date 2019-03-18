import {Component} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {UserLoginRequest} from '../../../model/user.model';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {UserService} from '../../../service/user/user.service';
import {AccountComponent} from '../account.component';

@Component({
    selector: 'app-account-login',
    templateUrl: './login.html',
    styleUrls: ['./login.scss']
})
export class LoginComponent extends AccountComponent {

    user: UserLoginRequest;
    redirect: string;

    constructor(private _userService: UserService, private _router: Router,
        _authStore: AuthentificationStore, private _route: ActivatedRoute) {
        super(_authStore);
        this.user = new UserLoginRequest();

        this._route.queryParams.subscribe(queryParams => {
           this.redirect = queryParams.redirect;
           this.user.request_token = queryParams.request;
        });
    }

    signIn() {
        this._userService.login(this.user).subscribe(() => {
            if (this.redirect) {
                this._router.navigateByUrl(decodeURIComponent(this.redirect));
            } else {
                this._router.navigate(['home']);
            }
        });
    }

    navigateToSignUp() {
        this._router.navigate(['/account/signup']);
    }

    navigateToPassword() {
        this._router.navigate(['/account/password']);
    }
}
