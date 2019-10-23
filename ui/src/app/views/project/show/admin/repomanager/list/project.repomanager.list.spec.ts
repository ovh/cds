import { HttpClientTestingModule } from '@angular/common/http/testing';
import { CUSTOM_ELEMENTS_SCHEMA, Injector } from '@angular/core';
import { getTestBed, TestBed } from '@angular/core/testing';
import { XHRBackend } from '@angular/http';
import { MockBackend } from '@angular/http/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ToasterService } from 'angular2-toaster/angular2-toaster';
import { Project } from 'app/model/project.model';
import { RepositoriesManager } from 'app/model/repositories.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { MonitoringService } from 'app/service/monitoring/monitoring.service';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { UserService } from 'app/service/user/user.service';
import { VariableService } from 'app/service/variable/variable.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { SharedModule } from 'app/shared/shared.module';
import { ToastService } from 'app/shared/toast/ToastService';
import { DisconnectRepositoryManagerInProject } from 'app/store/project.action';
import { NgxsStoreModule } from 'app/store/store.module';
import { ProjectModule } from 'app/views/project/project.module';
import { of } from 'rxjs';
import { ProjectRepoManagerComponent } from './project.repomanager.list.component';
describe('CDS: Project RepoManager List Component', () => {

    let injector: Injector;
    let backend: MockBackend;
    let projectStore: ProjectStore;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService,
                ToasterService,
                TranslateService,
                MonitoringService,
                TranslateParser,
                NavbarService,
                WorkflowService,
                WorkflowRunService,
                { provide: ToastService, useClass: MockToast },
                UserService,
                AuthenticationService
            ],
            imports: [
                ProjectModule,
                SharedModule,
                NgxsStoreModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ],
            schemas: [
                CUSTOM_ELEMENTS_SCHEMA
            ]
        });
        injector = getTestBed();
        backend = injector.get(MockBackend);
        projectStore = injector.get(ProjectStore);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        projectStore = undefined;
    });


    it('it should delete a repo manager', () => {
        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectRepoManagerComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.project = <Project>{
            key: 'key1',
            permissions: {
                readable: true,
                writable: true,
                executable: true
            }
        };
        fixture.componentInstance.reposmanagers = [
            <RepositoriesManager>{ name: 'stash' }
        ];

        fixture.detectChanges(true);

        let store: Store = injector.get(Store);
        spyOn(store, 'dispatch').and.callFake(() => {
            return of(null);
        });

        let compiled = fixture.debugElement.nativeElement;
        compiled.querySelector('.ui.red.button').click();
        fixture.detectChanges(true);

        compiled.querySelector('.ui.red.button.active').click();
        fixture.detectChanges(true);

        // Confirm deletion because we cannot simulate click on global modal ( out of scope of the component)
        fixture.componentInstance.confirmDeletion(true);

        expect(store.dispatch).toHaveBeenCalledWith(new DisconnectRepositoryManagerInProject({
            projectKey: 'key1',
            repoManager: 'stash'
        }));
    });
});

class MockToast {
    success(title: string, msg: string) {

    }
}
