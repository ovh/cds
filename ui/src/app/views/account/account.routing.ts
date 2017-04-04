import {LoginComponent} from './login/login.component';
import {Routes, RouterModule} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';
import {PasswordComponent} from './password/password.component';
import {SignUpComponent} from './signup/signup.component';
import {VerifyComponent} from './verify/verify.component';

const routes: Routes = [
    {
        path: '',
        children : [
            { path: '', redirectTo: 'login', pathMatch: 'full' },
            { path: 'login', component: LoginComponent },
            { path: 'password', component: PasswordComponent },
            { path: 'signup', component: SignUpComponent },
            { path: 'verify/:username/:token', component: VerifyComponent }
        ]
    }
];

export const accountRouting: ModuleWithProviders = RouterModule.forChild(routes);
