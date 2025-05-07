import { ModuleWithProviders } from '@angular/core';
import { PreloadAllModules, RouterModule, Routes } from '@angular/router';
import { AppModule } from 'app/app.module';
import { APIConfigGuard } from './guard/api-config.guard';
import { AuthenticationGuard } from './guard/authentication.guard';
import { SearchComponent } from './views/search/search.component';
import { HomeComponent } from './views/home/home.component';

const routes: Routes = [
    {
        path: '',
        canActivateChild: [AuthenticationGuard, APIConfigGuard],
        children: [
            {
                path: '',
                component: HomeComponent,
                data: { title: 'Home' }
            },
            {
                path: 'search',
                component: SearchComponent,
                data: { title: 'Search' }
            },
            {
                path: 'project', loadChildren: () => import('app/views/project/project.module')
                    .then(m => m.ProjectModule), data: { title: 'Project' }
            },
            {
                path: 'settings', loadChildren: () => import('app/views/settings/settings.module')
                    .then(m => m.SettingsModule), data: { title: 'Settings' }
            },
            {
                path: 'admin', loadChildren: () => import('app/views/admin/admin.module')
                    .then(m => m.AdminModule), data: { title: 'Admin' }
            }
        ]
    },
    {
        path: 'auth', loadChildren: () => import('app/views/auth/auth.module')
            .then(m => m.AuthModule), data: { title: 'Auth' }
    }
];

export const routing: ModuleWithProviders<AppModule> = RouterModule.forRoot(routes, {
    initialNavigation: 'enabledNonBlocking',
    preloadingStrategy: PreloadAllModules
});
