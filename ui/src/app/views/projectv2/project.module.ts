import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { SharedModule } from 'app/shared/shared.module';
import { projectV2Routing } from './project.routing';
import { ProjectV2ShowComponent } from 'app/views/projectv2/project.component';
import { ProjectV2TopMenuComponent } from 'app/views/projectv2/top-menu/project.top.menu.component';
import { ProjectV2SidebarComponent } from 'app/views/projectv2/sidebar/sidebar.component';
import {
    ProjectV2RepositoryAddComponent
} from 'app/views/projectv2/vcs/repository/project.repository.add.component';
import {
    ProjectV2RepositoryShowComponent
} from 'app/views/projectv2/vcs/repository/show/project.repository.show.component';
import { ProjectV2WorkerModelShowComponent } from "./vcs/repository/workermodel/show/project.workermodel.show.component";

@NgModule({
    declarations: [
        ProjectV2RepositoryAddComponent,
        ProjectV2RepositoryShowComponent,
        ProjectV2ShowComponent,
        ProjectV2SidebarComponent,
        ProjectV2TopMenuComponent,
        ProjectV2WorkerModelShowComponent
    ],
    imports: [
        SharedModule,
        RouterModule,
        projectV2Routing,
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ]
})
export class ProjectV2Module { }
