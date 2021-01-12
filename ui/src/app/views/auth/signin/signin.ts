import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthDriverManifest } from 'app/model/authentication.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { finalize } from 'rxjs/operators';
import * as zxcvbn from 'zxcvbn';

@Component({
    selector: 'app-auth-signin',
    templateUrl: './signin.html',
    styleUrls: ['./signin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SigninComponent implements OnInit {
    loading: boolean;
    redirect: string;
    mfa: boolean;

    isFirstConnection: boolean;
    localDriver: AuthDriverManifest;
    ldapDriver: AuthDriverManifest;
    externalDrivers: Array<AuthDriverManifest>;
    showSuccessSignup: boolean;
    localSigninActive: boolean;
    localSignupActive: boolean;
    ldapSigninActive: boolean;
    passwordError: string;
    passwordLevel: number;

    constructor(
        private _authenticationService: AuthenticationService,
        private _router: Router,
        private _route: ActivatedRoute,
        private _cd: ChangeDetectorRef
    ) {
        this.loading = true;
        this._route.queryParams.subscribe(queryParams => {
            this.redirect = queryParams.redirect;
            this.mfa = false;
            this._cd.markForCheck();
        });
    }

    ngOnInit() {
        this._authenticationService.getDrivers()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe((data) => {
                this.isFirstConnection = data.is_first_connection;
                this.localDriver = data.manifests.find(d => d.type === 'local');
                this.ldapDriver = data.manifests.find(d => d.type === 'ldap');
                this.externalDrivers = data.manifests
                    .filter(d => d.type !== 'local' && d.type !== 'ldap' && d.type !== 'builtin')
                    .sort((a, b) => a.type < b.type ? -1 : 1)
                    .map(d => {
                        switch (d.type) {
                            case 'corporate-sso': {
                                d.icon = 'shield alternate';
                                break;
                            }
                            case 'openid-connect': {
                                d.icon = 'openid';
                                break;
                            }
                            default: {
                                d.icon = d.type;
                                break;
                            }
                        }
                        return d;
                    });

                if (this.localDriver && this.isFirstConnection) {
                    this.localSignupActive = true;
                } else if (this.localDriver) {
                    this.localSigninActive = true;
                } else if (this.ldapDriver) {
                    this.ldapSigninActive = true;
                }
            });
    }

    clickShowLocalSignin() {
        this.localSigninActive = true;
        this.localSignupActive = false;
        this.ldapSigninActive = false;
        this._cd.markForCheck();
    }

    clickShowLocalSignup() {
        this.passwordError = null;
        this.passwordLevel = null;
        this.showSuccessSignup = false;
        this.localSigninActive = false;
        this.localSignupActive = true;
        this.ldapSigninActive = false;
        this._cd.markForCheck();
    }

    clickShowLDAPSignin() {
        this.localSigninActive = false;
        this.localSignupActive = false;
        this.ldapSigninActive = true;
        this._cd.markForCheck();
    }

    signup(f: NgForm) {
        if (f.value.password.length > 256) {
            this.passwordError = 'auth_password_too_long';
            this._cd.markForCheck();
            return;
        }
        if (this.passwordLevel < 3) {
            this.passwordError = 'auth_password_too_weak';
            this._cd.markForCheck();
            return;
        }

        this._authenticationService.localSignup(
            f.value.fullname,
            f.value.email,
            f.value.username,
            f.value.password,
            f.value.init_token
        ).subscribe(() => {
            this.showSuccessSignup = true;
            this._cd.markForCheck();
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

    onChangeSignupPassword(e: any) {
        this.passwordError = null;
        if (e.target.value.length <= 256) {
            let res = zxcvbn(e.target.value);
            this.passwordLevel = res.score;
        }
        this._cd.markForCheck();
    }
}
