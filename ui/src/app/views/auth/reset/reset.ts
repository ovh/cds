import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthenticationService } from 'app/service/services.module';
import { finalize } from 'rxjs/operators';

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
                this._cd.detectChanges();
                return;
            }

            this.token = token;
        });
    }

    resetPassword(f: NgForm) {
        this.loading = true;
        this.showSuccessMessage = false;
        this.showErrorMessage = false;
        this._authenticationService.localReset(this.token, f.value.password)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.detectChanges();
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
}
