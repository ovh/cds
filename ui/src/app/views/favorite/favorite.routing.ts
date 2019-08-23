import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { FavoriteComponent } from './favorite.component';

const routes: Routes = [
    {
        path: '',
        component: FavoriteComponent,
        canActivate: [AuthenticationGuard]
    }
];

export const favoriteRouting: ModuleWithProviders = RouterModule.forChild(routes);
