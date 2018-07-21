import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Params, Router} from '@angular/router';
import {cloneDeep} from 'lodash';
import {User} from '../../../model/user.model';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {UserService} from '../../../service/user/user.service';
import {AccountComponent} from '../account.component';

@Component({
    selector: 'app-account-verify',
    templateUrl: './verify.html',
    styleUrls: ['./verify.scss']
})
export class VerifyComponent extends AccountComponent implements OnInit  {

    userVerified: any;
    userVerifiedDisplay: any;
    showErrorMessage = false;

    constructor(private _userService: UserService, private _router: Router,
        private _activatedRoute: ActivatedRoute, _authStore: AuthentificationStore) {
        super(_authStore);
    }

    ngOnInit(): void {
        let params: Params = this._activatedRoute.snapshot.params;
        if (params['username'] && params['token']) {
            this._userService.verify(params['username'], params['token']).subscribe( res => {
                this.userVerified = res;
                this.userVerifiedDisplay = cloneDeep(res);
                delete this.userVerifiedDisplay.user.permissions;
            }, () => {
                this.showErrorMessage = true;
            });
        }
    }

    signIn() {
        let user: User = this.userVerified.user;
        user.password = this.userVerified.password;
        this._userService.login(user).subscribe(() => {
            this._router.navigate(['home']);
        });
    }
}
