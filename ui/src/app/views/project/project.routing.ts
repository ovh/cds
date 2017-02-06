import { ModuleWithProviders } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import {ProjectShowComponent} from './show/project.component';
import {ProjectAddComponent} from './add/project.add.component';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';

const projectRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: ProjectAddComponent },
            { path: ':key', component: ProjectShowComponent },
            { path: ':key/application', loadChildren: 'app/views/project/application/application.module#ApplicationModule'},
            { path: ':key/pipeline', loadChildren: 'app/views/project/pipeline/pipeline.module#PipelineModule'}
        ]
    }
];

export const projectRouting: ModuleWithProviders = RouterModule.forChild(projectRoutes);
