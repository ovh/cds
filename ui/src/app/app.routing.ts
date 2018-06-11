import {Routes, RouterModule, PreloadAllModules} from '@angular/router';
import {ModuleWithProviders} from '@angular/core';

const routes: Routes = [
    { path: '', redirectTo: 'home', pathMatch: 'full' },
    { path: 'home', loadChildren: 'app/views/home/home.module#HomeModule', data: { title: 'CDS - Home' }},
    { path: 'favorite', loadChildren: 'app/views/favorite/favorite.module#FavoriteModule', data: { title: 'CDS - Bookmarks' } },
    { path: 'broadcast', loadChildren: 'app/views/broadcast/broadcast.module#BroadcastModule', data: { title: 'CDS - Broadcasts' } },
    { path: 'account', loadChildren: 'app/views/account/account.module#AccountModule', data: { title: 'CDS - Account' } },
    { path: 'project', loadChildren: 'app/views/project/project.module#ProjectModule', data: { title: 'CDS - Project' } },
    { path: 'settings', loadChildren: 'app/views/settings/settings.module#SettingsModule', data: { title: 'CDS - Settings' } },
    { path: 'admin', loadChildren: 'app/views/admin/admin.module#AdminModule', data: { title: 'CDS - Admin' }}
];

export const routing: ModuleWithProviders = RouterModule.forRoot(routes, {
    initialNavigation: true,
    preloadingStrategy: PreloadAllModules
});
