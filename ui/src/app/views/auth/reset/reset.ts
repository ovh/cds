import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { finalize } from 'rxjs/operators';
import * as zxcvbn from 'zxcvbn';

@Component({
    selector: 'app-auth-reset',
    templateUrl: './reset.html',
    styleUrls: ['./reset.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ResetComponent implements OnInit {
    loading: boolean;
    showSuccessMessage: boolean;
    showErrorMessage: boolean;
    token: string;
    passwordError: string;
    passwordLevel: number;

    constructor(
        private _authenticationService: AuthenticationService,
        private _route: ActivatedRoute,
        private _router: Router,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        this._route.queryParams.subscribe(queryParams => {
            let token = queryParams['token'];
            if (!token) {
                this.showErrorMessage = true;
                this._cd.markForCheck();
                return;
            }

            this.token = token;
        });
    }

    resetPassword(f: NgForm) {
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

        this.loading = true;
        this.showSuccessMessage = false;
        this.showErrorMessage = false;
        this._authenticationService.localReset(this.token, f.value.password)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(res => {
                this.showSuccessMessage = true;
            }, () => {
                this.showErrorMessage = true;
            });
    }

    navigateToHome() {
        this._router.navigate(['/']);
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
