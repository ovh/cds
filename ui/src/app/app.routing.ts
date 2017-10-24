import {Routes, RouterModule, PreloadAllModules} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';

const routes: Routes = [
    { path: '', redirectTo: 'home', pathMatch: 'full' },
    { path: 'home', loadChildren: 'app/views/home/home.module#HomeModule' },
    { path: 'account', loadChildren: 'app/views/account/account.module#AccountModule' },
    { path: 'project', loadChildren: 'app/views/project/project.module#ProjectModule' },
    { path: 'settings', loadChildren: 'app/views/settings/settings.module#SettingsModule' },
    { path: 'warnings', loadChildren: 'app/views/warnings/warnings.module#WarningsModule' },
    { path: 'admin', loadChildren: 'app/views/admin/admin.module#AdminModule'}
];

export const routing: ModuleWithProviders = RouterModule.forRoot(routes, {
    initialNavigation: true,
    preloadingStrategy: PreloadAllModules
});
