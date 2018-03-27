import { ModuleWithProviders } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import {ProjectShowComponent} from './show/project.component';
import {ProjectListComponent} from './list/project.list.component';
import {ProjectAddComponent} from './add/project.add.component';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';

const projectRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: ProjectAddComponent },
            { path: 'list/all', component: ProjectListComponent },
            { path: ':key', component: ProjectShowComponent },
            { path: ':key/workflow', loadChildren: 'app/views/workflow/workflow.module#WorkflowModule'},
            { path: ':key/application', loadChildren: 'app/views/application/application.module#ApplicationModule'},
            { path: ':key/pipeline', loadChildren: 'app/views/pipeline/pipeline.module#PipelineModule'}
        ]
    }
];

export const projectRouting: ModuleWithProviders = RouterModule.forChild(projectRoutes);
