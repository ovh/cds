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
            {
                path: 'login',
                component: LoginComponent,
                data: { title: 'CDS - Login' }
            },
            { path: 'password', component: PasswordComponent, data: { title: 'CDS - Reset Password' }},
            { path: 'signup', component: SignUpComponent, data: { title: 'CDS - Signup' }},
            { path: 'verify/:username/:token', component: VerifyComponent }
        ]
    }
];

export const accountRouting: ModuleWithProviders = RouterModule.forChild(routes);
