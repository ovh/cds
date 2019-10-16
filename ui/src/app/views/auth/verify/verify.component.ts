import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthentifiedUser } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { jws } from 'jsrsasign';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-auth-verify',
    templateUrl: './verify.html',
    styleUrls: ['./verify.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VerifyComponent implements OnInit {
    userVerified: any;
    userVerifiedDisplay: any;
    showErrorMessage: boolean;
    showSuccessMessage: boolean;
    loading: boolean;
    user: AuthentifiedUser;
    payloadData: any;
    showInitTokenForm: boolean;
    token: string;

    constructor(
        private _router: Router,
        private _route: ActivatedRoute,
        private _authenticationService: AuthenticationService,
        private _cd: ChangeDetectorRef
    ) {
        this.loading = true;
    }

    ngOnInit(): void {
        this._route.queryParams.subscribe(queryParams => {
            this.token = queryParams['token'];
            if (!this.token) {
                this.showErrorMessage = true;
                this.loading = false;
                this._cd.markForCheck();
                return;
            }

            // Parse JWS state
            let payload = jws.JWS.parse(this.token).payloadObj;
            this.payloadData = payload.data;

            // If the first connection flag is set, show init token form
            if (this.payloadData && this.payloadData.is_first_connection) {
                this.loading = false;
                this.showInitTokenForm = true;
                this._cd.markForCheck();
                return;
            }

            this.sendVerifyRequest();
        });
    }

    navigateToSignin(): void {
        this._router.navigate(['/auth/signin']);
    }

    navigateToHome(): void {
        this._router.navigate(['/']);
    }

    verify(f: NgForm): void {
        this.sendVerifyRequest(f.value.init_token);
    }

    sendVerifyRequest(initToken?: string): void {
        this.showInitTokenForm = false;
        this.loading = true;
        this._cd.markForCheck();
        this._authenticationService.localVerify(this.token, initToken)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(res => {
                this.user = res.user;
                this.showSuccessMessage = true;
            }, () => {
                this.showErrorMessage = true;
            });
    }
}
