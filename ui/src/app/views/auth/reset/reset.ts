import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { User } from 'app/model/user.model';
import { AuthentificationStore } from '../../../service/authentication/authentification.store';
import { UserService } from '../../../service/user/user.service';

@Component({
    selector: 'app-auth-reset',
    templateUrl: './reset.html',
    styleUrls: ['./reset.scss'],
})
export class ResetComponent {
    user: User;
    showWaitingMessage = false;

    constructor(
        private _userService: UserService,
        private _router: Router,
        _authStore: AuthentificationStore
    ) {
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
        this._router.navigate(['/auth/signin']);
    }

    navigateToLogin() {
        this._router.navigate(['/auth/signin']);
    }
}
