import { ModuleWithProviders } from '@angular/core';
import { PreloadAllModules, RouterModule, Routes } from '@angular/router';
import { AppModule } from 'app/app.module';

const routes: Routes = [
    { path: '', redirectTo: 'home', pathMatch: 'full' },
    {
        path: 'home', loadChildren: () => import('app/views/home/home.module')
            .then(m => m.HomeModule), data: { title: 'Home' }
    },
    {
        path: 'favorite', loadChildren: () => import('app/views/favorite/favorite.module')
            .then(m => m.FavoriteModule), data: { title: 'Bookmarks' }
    },
    {
        path: 'broadcast', loadChildren: () => import('app/views/broadcast/broadcast.module')
            .then(m => m.BroadcastModule), data: { title: 'Broadcasts' }
    },
    {
        path: 'auth', loadChildren: () => import('app/views/auth/auth.module')
            .then(m => m.AuthModule), data: { title: 'Auth' }
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
];

export const routing: ModuleWithProviders<AppModule> = RouterModule.forRoot(routes, {
    initialNavigation: 'enabledNonBlocking',
    preloadingStrategy: PreloadAllModules,
    relativeLinkResolution: 'legacy'
});
