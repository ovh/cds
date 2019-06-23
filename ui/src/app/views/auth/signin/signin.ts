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
    redirect: string;
    apiURL: string;

    localDriver: AuthDriverManifest;
    ldapDriver: AuthDriverManifest;
    externalDrivers: Array<AuthDriverManifest>;

    constructor(
        private _authenticationService: AuthenticationService,
        private _router: Router,
        private _route: ActivatedRoute
    ) {
        this._route.queryParams.subscribe(queryParams => {
            this.redirect = queryParams.redirect;
        });
    }

    ngOnInit() {
        this._authenticationService.getDrivers().subscribe((ds) => {
            this.localDriver = ds.find(d => d.type === 'local');
            this.ldapDriver = ds.find(d => d.type === 'ldap');
            this.externalDrivers = ds.filter(d => d.type !== 'local' && d.type !== 'ldap');
        });
    }


    signup(f: NgForm) {
        this._authenticationService.localSignup(
            f.value.fullname,
            f.value.email,
            f.value.username,
            f.value.password
        ).subscribe(() => {
            // TODO show successfull signup, you will receive an email
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

    navigateToAskReset() {
        this._router.navigate(['/auth/ask-reset']);
    }
}
