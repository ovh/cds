import { ModuleWithProviders } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { ProjectV2Module } from 'app/views/projectv2/project.module';
import { ProjectV2ShowComponent } from 'app/views/projectv2/project.component';
import { FeatureGuard } from 'app/guard/feature.guard';
import { FeatureNames } from 'app/service/feature/feature.service';
import { ProjectV2RepositoryAddComponent } from 'app/views/projectv2/vcs/repository/project.repository.add.component';
import { Projectv2Resolver } from 'app/service/project/project.resolver';
import {
    ProjectV2RepositoryShowComponent
} from 'app/views/projectv2/vcs/repository/show/project.repository.show.component';
import { ProjectV2WorkerModelShowComponent } from "./vcs/repository/workermodel/show/project.workermodel.show.component";
import { ProjectV2ActionShowComponent } from "./vcs/repository/action/show/project.action.show.component";
import { ProjectV2WorkflowShowComponent } from "./vcs/repository/workflow/show/project.workflow.show.component";
import { ProjectV2WorkflowRunComponent } from "./run/project.run.component";
import { ProjectV2WorkflowRunListComponent } from './run-list/run-list.component';


const projectRoutes: Routes = [
    {
        path: ':key',
        component: ProjectV2ShowComponent,
        canActivate: [FeatureGuard],
        data: {
            title: '{key} • Project', feature: FeatureNames.AllAsCode
        },
        children: [
            {
                path: '', redirectTo: 'explore', pathMatch: 'full'
            },
            {
                path: 'explore',
                children: [
                    {
                        path: 'vcs/:vcsName/repository',
                        component: ProjectV2RepositoryAddComponent,
                        data: { title: 'Add • Repository' },
                        resolve: {
                            project: Projectv2Resolver,
                        }
                    },
                    {
                        path: 'vcs/:vcsName/repository/:repoName',
                        component: ProjectV2RepositoryShowComponent,
                        data: { title: '{repoName} • Repository' },
                        resolve: {
                            project: Projectv2Resolver,
                        }
                    },
                    {
                        path: 'vcs/:vcsName/repository/:repoName/workermodel/:workerModelName',
                        component: ProjectV2WorkerModelShowComponent,
                        data: { title: '{workerModelName} • Worker Model' },
                        resolve: {
                            project: Projectv2Resolver,
                        },
                    },
                    {
                        path: 'vcs/:vcsName/repository/:repoName/action/:actionName',
                        component: ProjectV2ActionShowComponent,
                        data: { title: '{actionName} • Action' },
                        resolve: {
                            project: Projectv2Resolver,
                        },
                    },
                    {
                        path: 'vcs/:vcsName/repository/:repoName/workflow/:workflowName',
                        component: ProjectV2WorkflowShowComponent,
                        data: { title: '{workflowName} • Workflow' },
                        resolve: {
                            project: Projectv2Resolver,
                        },
                    }
                ]
            },
            {
                path: 'run',
                component: ProjectV2WorkflowRunListComponent,
                data: { title: 'List • Workflow Runs' },
                resolve: {
                    project: Projectv2Resolver,
                }
            },
            {
                path: 'run/vcs/:vcsName/repository/:repoName/workflow/:workflowName',
                component: ProjectV2WorkflowRunComponent,
                data: { title: '{workflowName} • Workflow Run' },
                resolve: {
                    project: Projectv2Resolver,
                }
            }
        ]
    }
];

export const projectV2Routing: ModuleWithProviders<ProjectV2Module> = RouterModule.forChild(projectRoutes);
