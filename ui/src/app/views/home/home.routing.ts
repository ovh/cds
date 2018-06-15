import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {HomeComponent} from './home.component';

const routes: Routes = [
    {
        path: '',
        component: HomeComponent,
        canActivate: [CanActivateAuthRoute]
    }
];

export const homeRouting: ModuleWithProviders = RouterModule.forChild(routes);
