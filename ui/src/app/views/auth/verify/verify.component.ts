import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { AuthenticationService } from 'app/service/services.module';
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
    loading: boolean;

    constructor(
        private _router: Router,
        private _activatedRoute: ActivatedRoute,
        private _authenticationService: AuthenticationService,
        private _cd: ChangeDetectorRef
    ) {
        this.loading = true;
    }

    ngOnInit(): void {
        let params: Params = this._activatedRoute.snapshot.params;

        let token = params['token'];
        if (!token) {
            this.showErrorMessage = true;
            this.loading = false;
            this._cd.detectChanges();
            return;
        }

        this._authenticationService.localVerify(token)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.detectChanges();
            }))
            .subscribe(res => {
                // TODO store token then redirect
                // this._router.navigate(['home']);
            }, () => {
                this.showErrorMessage = true;
            });
    }

    navigateToSignin() {
        this._router.navigate(['/auth/signin']);
    }
}
