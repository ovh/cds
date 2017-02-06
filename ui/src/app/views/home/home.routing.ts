import {Routes, RouterModule} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';
import {HomeComponent} from './home.component';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';

const routes: Routes = [
    {
        path: '',
        component: HomeComponent,
        canActivate: [CanActivateAuthRoute]
    }
];

export const homeRouting: ModuleWithProviders = RouterModule.forChild(routes);
