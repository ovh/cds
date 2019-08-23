import { Component, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthDriverManifest } from 'app/model/authentication.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';

@Component({
    selector: 'app-auth-signin',
    templateUrl: './signin.html',
    styleUrls: ['./signin.scss']
})
export class SigninComponent implements OnInit {
    loading: boolean;
    redirect: string;
    mfa: boolean;
    apiURL: string;

    isFirstConnection: boolean;
    localDriver: AuthDriverManifest;
    ldapDriver: AuthDriverManifest;
    externalDrivers: Array<AuthDriverManifest>;
    showSuccessSignup: boolean;

    constructor(
        private _authenticationService: AuthenticationService,
        private _router: Router,
        private _route: ActivatedRoute
    ) {
        this.loading = true;
        this._route.queryParams.subscribe(queryParams => {
            this.redirect = queryParams.redirect;
            this.mfa = false;
        });
    }

    ngOnInit() {
        this._authenticationService.getDrivers().subscribe((data) => {
            this.isFirstConnection = data.is_first_connection;
            this.localDriver = data.manifests.find(d => d.type === 'local');
            this.ldapDriver = data.manifests.find(d => d.type === 'ldap');
            this.externalDrivers = data.manifests
                .filter(d => d.type !== 'local' && d.type !== 'ldap' && d.type !== 'builtin')
                .sort((a, b) => a.type < b.type ? -1 : 1)
                .map(d => {
                    d.icon = d.type === 'corporate-sso' ? 'shield alternate' : d.type;
                    return d;
                });
            this.loading = false;
        });
    }

    resetSignup() {
        this.showSuccessSignup = false;
    }

    signup(f: NgForm) {
        this._authenticationService.localSignup(
            f.value.fullname,
            f.value.email,
            f.value.username,
            f.value.password,
            f.value.init_token
        ).subscribe(() => {
            this.showSuccessSignup = true;
        });
    }

    signin(f: NgForm) {
        this._authenticationService.localSignin(f.value.username, f.value.password).subscribe(() => {
            if (this.redirect) {
                this._router.navigateByUrl(decodeURIComponent(this.redirect));
            } else {
                this._router.navigate(['home']);
            }
        });
    }

    ldapSignin(f: NgForm) {
        this._authenticationService.ldapSignin(f.value.bind, f.value.password, f.value.init_token).subscribe(() => {
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
