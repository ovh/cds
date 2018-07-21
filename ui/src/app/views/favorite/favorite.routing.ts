import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {FavoriteComponent} from './favorite.component';

const routes: Routes = [
    {
        path: '',
        component: FavoriteComponent,
        canActivate: [CanActivateAuthRoute]
    }
];

export const favoriteRouting: ModuleWithProviders = RouterModule.forChild(routes);
