import { ChangeDetectionStrategy, ChangeDetectorRef, Component } from '@angular/core';
import { NgForm } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { finalize } from 'rxjs/operators';

@Component({
  selector: 'app-auth-ask-reset',
  templateUrl: './ask-reset.html',
  styleUrls: ['./ask-reset.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class AskResetComponent {
  loading: boolean;
  showSuccessMessage: boolean;
  showErrorMessage: boolean;

  constructor(
    private _authenticationService: AuthenticationService,
    private _router: Router,
    private _cd: ChangeDetectorRef
  ) { }

  askReset(f: NgForm) {
    this.loading = true;
    this.showSuccessMessage = false;
    this.showErrorMessage = false;
    this._authenticationService.localAskReset(f.value.email)
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

  navigateToSignin() {
    this._router.navigate(['/auth/signin']);
  }
}
