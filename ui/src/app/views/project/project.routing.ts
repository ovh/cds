import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { ProjectModule } from 'app/views/project/project.module';
import { ProjectAddComponent } from './add/project.add.component';
import { ProjectShowComponent } from './show/project.component';
import { ProjectComponent } from './project.component';
import { ProjectV2ExploreComponent } from '../projectv2/explore/explore.component';
import { ProjectV2RunListComponent } from '../projectv2/run-list/run-list.component';
import { ProjectV2RunComponent } from '../projectv2/run/run.component';
import { ProjectSettingsComponent } from './settings/settings.component';
import { ProjectV2ExploreEntityComponent } from '../projectv2/explore/explore-entity.component';
import { ProjectV2ExploreRepositoryAddComponent } from '../projectv2/explore/explore-repository-add.component';
import { ProjectV2ExploreRepositoryComponent } from '../projectv2/explore/explore-repository.component';
import { ProjectExistsGuard, ProjectGuard, ProjectV2Guard } from 'app/views/project/project.guard';

const projectRoutes: Routes = [
    {
        path: '',
        children: [
            {
                path: '', redirectTo: '/search?type=project', pathMatch: 'full'
            },
            { path: 'add', component: ProjectAddComponent, data: { title: 'Add • Project' } },
            {
                path: ':key',
                canActivate: [ProjectGuard],
                component: ProjectComponent,
                children: [
                    {
                        path: '',
                        canActivate: [ProjectExistsGuard],
                        component: ProjectShowComponent,
                        data: { title: '{key} • Project' },
                    },
                    {
                        path: 'settings',
                        canActivate: [ProjectV2Guard],
                        component: ProjectSettingsComponent,
                        data: { title: '{key} • Settings' },
                    },
                    {
                        path: 'workflow', loadChildren:
                            () => import('app/views/workflow/workflow.module').then(m => m.WorkflowModule)
                    },
                    {
                        path: 'environment', loadChildren:
                            () => import('app/views/environment/environment.module').then(m => m.EnvironmentModule)
                    },
                    {
                        path: 'application', loadChildren:
                            () => import('app/views/application/application.module').then(m => m.ApplicationModule)
                    },
                    {
                        path: 'pipeline', loadChildren:
                            () => import('app/views/pipeline/pipeline.module').then(m => m.PipelineModule)
                    },
                    {
                        path: 'explore',
                        canActivate: [ProjectV2Guard],
                        component: ProjectV2ExploreComponent,
                        children: [
                            {
                                path: 'vcs/:vcsName/repository',
                                component: ProjectV2ExploreRepositoryAddComponent,
                                data: { title: 'Add • Repository' }
                            },
                            {
                                path: 'vcs/:vcsName/repository/:repoName',
                                children: [
                                    {
                                        path: '', redirectTo: 'settings', pathMatch: 'full'
                                    },
                                    {
                                        path: 'settings',
                                        component: ProjectV2ExploreRepositoryComponent,
                                        data: { title: '{repoName} • Repository' }
                                    }
                                ]
                            },
                            {
                                path: 'vcs/:vcsName/repository/:repoName/:entityType/:entityName',
                                component: ProjectV2ExploreEntityComponent,
                                data: { title: '{entityName} • Entity' }
                            }
                        ]
                    },
                    {
                        path: 'run',
                        canActivate: [ProjectV2Guard],
                        children: [
                            {
                                path: '',
                                component: ProjectV2RunListComponent,
                                data: { title: 'List • Workflow Runs' }
                            },
                            {
                                path: ':workflowRunID',
                                component: ProjectV2RunComponent,
                                data: { title: '{workflowRunID} • Workflow Run' }
                            }
                        ]
                    }
                ]
            }
        ]
    }
];

export const projectRouting: ModuleWithProviders<ProjectModule> = RouterModule.forChild(projectRoutes);
