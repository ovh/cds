import { ModuleWithProviders } from '@angular/core';
import { CanMatchFn, RouterModule, Routes, UrlSegment } from '@angular/router';
import { ProjectModule } from 'app/views/project/project.module';
import { ProjectAddComponent } from './add/project.add.component';
import { ProjectShowComponent } from './show/project.component';
import { ProjectComponent } from './project.component';
import { ProjectV2ExploreComponent } from '../projectv2/explore/explore.component';
import { ProjectV2RunListComponent } from '../projectv2/run-list/run-list.component';
import { ProjectV2RunComponent } from '../projectv2/run/run.component';
import { ProjectSettingsComponent } from './settings/settings.component';
import { ProjectExistsGuard, ProjectGuard, ProjectV2Guard } from 'app/views/project/project.guard';
import { ProjectV2ExploreOverviewComponent } from '../projectv2/explore/explore-overview.component';
import { ProjectV2RepositoryComponent } from '../projectv2/explore/repository/repository.component';
import { ProjectV2RepositoryEntitiesComponent } from '../projectv2/explore/repository/repository-entities.component';
import { ProjectV2RepositoryActivityComponent } from '../projectv2/explore/repository/repository-activity.component';
import { ProjectV2RepositoryAnalysesComponent } from '../projectv2/explore/repository/repository-analyses.component';
import { ProjectV2RepositorySettingsComponent } from '../projectv2/explore/repository/repository-settings.component';
import { EntityTypeUtil } from 'app/model/entity.model';

/** Only a known entity type takes the ':entityType' segment, so that the repository tabs keep theirs. */
export const entityTypeCanMatch: CanMatchFn = (_, segments: UrlSegment[]): boolean => {
    try {
        EntityTypeUtil.fromURLParam(segments[0]?.path);
        return true;
    } catch (e) {
        return false;
    }
};

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
                                path: '',
                                component: ProjectV2ExploreOverviewComponent,
                                data: { title: 'Explore' }
                            },
                            {
                                path: 'vcs/:vcsName/repository/:repoName',
                                component: ProjectV2RepositoryComponent,
                                data: { title: '{repoName} • Repository' },
                                children: [
                                    {
                                        path: 'activity',
                                        component: ProjectV2RepositoryActivityComponent,
                                        data: { title: '{repoName} • Activity' }
                                    },
                                    {
                                        path: 'analyses',
                                        component: ProjectV2RepositoryAnalysesComponent,
                                        data: { title: '{repoName} • Analyses' }
                                    },
                                    {
                                        path: 'settings',
                                        component: ProjectV2RepositorySettingsComponent,
                                        data: { title: '{repoName} • Settings' }
                                    },
                                    {
                                        path: ':entityType',
                                        canMatch: [entityTypeCanMatch],
                                        component: ProjectV2RepositoryEntitiesComponent,
                                        data: { title: '{repoName} • Repository' }
                                    },
                                    {
                                        path: ':entityType/:entityName',
                                        canMatch: [entityTypeCanMatch],
                                        component: ProjectV2RepositoryEntitiesComponent,
                                        data: { title: '{entityName} • Entity' }
                                    }
                                ]
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
                                component: ProjectV2RunComponent
                            }
                        ]
                    }
                ]
            }
        ]
    }
];

export const projectRouting: ModuleWithProviders<ProjectModule> = RouterModule.forChild(projectRoutes);
