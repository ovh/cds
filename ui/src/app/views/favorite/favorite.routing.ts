import {Routes, RouterModule} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';
import {FavoriteComponent} from './favorite.component';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';

const routes: Routes = [
    {
        path: '',
        component: FavoriteComponent,
        canActivate: [CanActivateAuthRoute]
    }
];

export const favoriteRouting: ModuleWithProviders = RouterModule.forChild(routes);
