import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { FeatureGuard } from 'app/guard/feature.guard';
import { FeatureNames } from 'app/service/feature/feature.service';
import { ProjectModule } from 'app/views/project/project.module';
import { ProjectAddComponent } from './add/project.add.component';
import { ProjectListComponent } from './list/project.list.component';
import { ProjectShowComponent } from './show/project.component';
import { ProjectComponent } from './project.component';
import { ProjectV2ExploreComponent } from '../projectv2/explore/explore.component';
import { ProjectV2RepositoryAddComponent } from '../projectv2/explore/vcs/repository/project.repository.add.component';
import { ProjectV2RepositoryShowComponent } from '../projectv2/explore/vcs/repository/show/project.repository.show.component';
import { ProjectV2WorkerModelShowComponent } from '../projectv2/explore/vcs/repository/workermodel/show/project.workermodel.show.component';
import { ProjectV2ActionShowComponent } from '../projectv2/explore/vcs/repository/action/show/project.action.show.component';
import { ProjectV2WorkflowShowComponent } from '../projectv2/explore/vcs/repository/workflow/show/project.workflow.show.component';
import { ProjectV2WorkflowRunListComponent } from '../projectv2/run-list/run-list.component';
import { ProjectV2WorkflowRunComponent } from '../projectv2/run/project.run.component';
import { Projectv2Resolver } from 'app/service/services.module';
import { ProjectSettingsComponent } from './settings/settings.component';

const projectRoutes: Routes = [
    {
        path: '',
        children: [
            { path: '', component: ProjectAddComponent, data: { title: 'Add • Project' } },
            { path: 'list/all', component: ProjectListComponent, data: { title: 'List • Project' } },
            {
                path: ':key',
                component: ProjectComponent,
                resolve: {
                    project: Projectv2Resolver,
                },
                children: [
                    {
                        path: '',
                        component: ProjectShowComponent,
                        data: { title: '{key} • Project' },
                    },
                    {
                        path: 'settings',
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
                        path: ':key/workflowv3', loadChildren:
                            () => import('app/views/workflowv3/workflowv3.module').then(m => m.WorkflowV3Module),
                        canActivate: [FeatureGuard],
                        data: { feature: FeatureNames.WorkflowV3 }
                    },
                    {
                        path: 'explore',
                        canActivate: [FeatureGuard],
                        data: { feature: FeatureNames.AllAsCode },
                        component: ProjectV2ExploreComponent,
                        children: [
                            {
                                path: 'vcs/:vcsName/repository',
                                component: ProjectV2RepositoryAddComponent,
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
                                        component: ProjectV2RepositoryShowComponent,
                                        data: { title: '{repoName} • Repository' }
                                    }
                                ]
                            },
                            {
                                path: 'vcs/:vcsName/repository/:repoName/workermodel/:workerModelName',
                                component: ProjectV2WorkerModelShowComponent,
                                data: { title: '{workerModelName} • Worker Model' }
                            },
                            {
                                path: 'vcs/:vcsName/repository/:repoName/action/:actionName',
                                component: ProjectV2ActionShowComponent,
                                data: { title: '{actionName} • Action' }
                            },
                            {
                                path: 'vcs/:vcsName/repository/:repoName/workflow/:workflowName',
                                component: ProjectV2WorkflowShowComponent,
                                data: { title: '{workflowName} • Workflow' }
                            }
                        ]
                    },
                    {
                        path: 'run',
                        canActivate: [FeatureGuard],
                        data: { feature: FeatureNames.AllAsCode },
                        children: [
                            {
                                path: '',
                                component: ProjectV2WorkflowRunListComponent,
                                data: { title: 'List • Workflow Runs' }
                            },
                            {
                                path: ':workflowRunID',
                                component: ProjectV2WorkflowRunComponent,
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
