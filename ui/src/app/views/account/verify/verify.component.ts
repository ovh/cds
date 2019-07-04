import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { UserLoginRequest } from 'app/model/user.model';
import { AuthentificationStore } from 'app/service/auth/authentification.store';
import { UserService } from 'app/service/user/user.service';
import cloneDeep from 'lodash-es/cloneDeep';
import { AccountComponent } from '../account.component';

@Component({
    selector: 'app-account-verify',
    templateUrl: './verify.html',
    styleUrls: ['./verify.scss']
})
export class VerifyComponent extends AccountComponent implements OnInit {

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
            this._userService.verify(params['username'], params['token']).subscribe(res => {
                this.userVerified = res;
                this.userVerifiedDisplay = cloneDeep(res);
                delete this.userVerifiedDisplay.user.permissions;
            }, () => {
                this.showErrorMessage = true;
            });
        }
    }

    signIn() {
        let userloginRequest = new UserLoginRequest();
        userloginRequest.username = this.userVerified.user.username;
        userloginRequest.password = this.userVerified.password;

        this._userService.login(userloginRequest).subscribe(() => {
            this._router.navigate(['home']);
        });
    }
}
