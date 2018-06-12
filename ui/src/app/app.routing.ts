import {Routes, RouterModule, PreloadAllModules} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';

const routes: Routes = [
    { path: '', redirectTo: 'home', pathMatch: 'full' },
    { path: 'home', loadChildren: 'app/views/home/home.module#HomeModule', data: { title: 'Home' }},
    { path: 'favorite', loadChildren: 'app/views/favorite/favorite.module#FavoriteModule', data: { title: 'Bookmarks' } },
    { path: 'broadcast', loadChildren: 'app/views/broadcast/broadcast.module#BroadcastModule', data: { title: 'Broadcasts' } },
    { path: 'account', loadChildren: 'app/views/account/account.module#AccountModule', data: { title: 'Account' } },
    { path: 'project', loadChildren: 'app/views/project/project.module#ProjectModule', data: { title: 'Project' } },
    { path: 'settings', loadChildren: 'app/views/settings/settings.module#SettingsModule', data: { title: 'Settings' } },
    { path: 'admin', loadChildren: 'app/views/admin/admin.module#AdminModule', data: { title: 'Admin' }}
];

export const routing: ModuleWithProviders = RouterModule.forRoot(routes, {
    initialNavigation: true,
    preloadingStrategy: PreloadAllModules
});
