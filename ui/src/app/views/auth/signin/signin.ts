import { Component } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { UserLoginRequest } from 'app/model/user.model';
import { AuthentificationStore } from 'app/service/authentication/authentification.store';
import { UserService } from 'app/service/user/user.service';
import { environment } from 'environments/environment';

@Component({
    selector: 'app-auth-signin',
    templateUrl: './signin.html',
    styleUrls: ['./signin.scss']
})
export class SigninComponent {
    user: UserLoginRequest;
    redirect: string;
    apiURL: string;

    constructor(
        private _userService: UserService,
        private _router: Router,
        _authStore: AuthentificationStore,
        private _route: ActivatedRoute
    ) {
        this.user = new UserLoginRequest();

        this._route.queryParams.subscribe(queryParams => {
            this.redirect = queryParams.redirect;
            this.user.request_token = queryParams.request;
        });

        this.apiURL = environment.apiURL;
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
        this._router.navigate(['/auth/signin']);
    }

    navigateToPassword() {
        this._router.navigate(['/auth/ask-reset']);
    }
}
