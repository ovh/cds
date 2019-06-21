import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthDriverManifest } from 'app/model/authentication.model';
import { UserLoginRequest } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { UserService } from 'app/service/user/user.service';
import { environment } from 'environments/environment';

@Component({
    selector: 'app-auth-signin',
    templateUrl: './signin.html',
    styleUrls: ['./signin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class SigninComponent implements OnInit {
    user: UserLoginRequest;
    redirect: string;
    apiURL: string;

    localDriver: AuthDriverManifest;
    ldapDriver: AuthDriverManifest;
    externalDrivers: Array<AuthDriverManifest>;

    constructor(
        private _userService: UserService,
        private _authenticationService: AuthenticationService,
        private _router: Router,
        private _route: ActivatedRoute,
        private _cd: ChangeDetectorRef
    ) {
        this.user = new UserLoginRequest();

        this._route.queryParams.subscribe(queryParams => {
            this.redirect = queryParams.redirect;
            this.user.request_token = queryParams.request;
        });

        this.apiURL = environment.apiURL;
    }

    ngOnInit() {
        this._authenticationService.getDrivers().subscribe((ds) => {
            this.localDriver = ds.find(d => d.type === 'local');
            this.ldapDriver = ds.find(d => d.type === 'ldap');
            this.externalDrivers = ds.filter(d => d.type !== 'local' && d.type !== 'ldap');
            this._cd.detectChanges();
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

    navigateToAskReset() {
        this._router.navigate(['/auth/ask-reset']);
    }
}
