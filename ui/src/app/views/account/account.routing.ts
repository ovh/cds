import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {LoginComponent} from './login/login.component';
import {PasswordComponent} from './password/password.component';
import {SignUpComponent} from './signup/signup.component';
import {VerifyComponent} from './verify/verify.component';

const routes: Routes = [
    {
        path: '',
        children : [
            { path: '', redirectTo: 'login', pathMatch: 'full' },
            {
                path: 'login',
                component: LoginComponent,
                data: { title: 'CDS • Login' }
            },
            { path: 'password', component: PasswordComponent, data: { title: 'Reset Password' }},
            { path: 'signup', component: SignUpComponent, data: { title: 'Signup' }},
            { path: 'verify/:username/:token', component: VerifyComponent }
        ]
    }
];

export const accountRouting: ModuleWithProviders = RouterModule.forChild(routes);
