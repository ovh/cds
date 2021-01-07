import { HttpClientTestingModule } from '@angular/common/http/testing';
import { CUSTOM_ELEMENTS_SCHEMA, Injector } from '@angular/core';
import { getTestBed, TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ToasterService } from 'angular2-toaster-sgu';
import { ApplicationService } from 'app/service/application/application.service';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { MonitoringService } from 'app/service/monitoring/monitoring.service';
import { RouterService } from 'app/service/router/router.service';
import { HelpService } from 'app/service/services.module';
import { UserService } from 'app/service/user/user.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AddProject } from 'app/store/project.action';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { Group, GroupPermission } from 'app/model/group.model';
import { Project } from 'app/model/project.model';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { GroupService } from 'app/service/group/group.service';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { VariableService } from 'app/service/variable/variable.service';
import { SharedModule } from 'app/shared/shared.module';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectModule } from '../project.module';
import { ProjectAddComponent } from './project.add.component';

describe('CDS: Project Show Component', () => {

    let injector: Injector;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            providers: [
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                WorkflowRunService,
                AuthenticationService,
                NavbarService,
                ProjectService,
                PipelineService,
                MonitoringService,
                ApplicationService,
                EnvironmentService,
                VariableService,
                ToasterService,
                HelpService,
                TranslateService,
                TranslateParser,
                GroupService,
                UserService,
                RouterService,
                WorkflowService,
                { provide: ToastService, useClass: MockToast }
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
        }).compileComponents();
        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });


    it('it should create a project', () => {
        let store: Store = injector.get(Store);
        let router: Router = injector.get(Router);

        spyOn(store, 'dispatch').and.callFake(() => of(null));

        spyOn(router, 'navigate').and.callFake(() => new Promise(() => true));

        // Create Project RepoManager Form Component
        let fixture = TestBed.createComponent(ProjectAddComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.project.name = 'FooProject';
        fixture.componentInstance.project.key = 'BAR';

        fixture.componentInstance.project.groups = new Array<GroupPermission>();
        fixture.componentInstance.group = new Group();
        fixture.componentInstance.group.name = 'foo';

        fixture.componentInstance.createProject();

        let project = new Project();
        project.name = 'FooProject';
        project.key = 'BAR';
        project.groups = new Array<GroupPermission>();
        project.groups.push(new GroupPermission());
        project.groups[0].group = new Group();
        project.groups[0].group.name = 'foo';
        project.groups[0].permission = 7;
        expect(store.dispatch).toHaveBeenCalledWith(new AddProject(project));
        expect(router.navigate).toHaveBeenCalled();
    });

    it('it should generate errors', () => {
        let fixture = TestBed.createComponent(ProjectAddComponent);
        fixture.componentInstance.createProject();

        expect(fixture.componentInstance.nameError).toBeTruthy();

        // pattern error
        fixture.componentInstance.createProject();
    });
});

class MockToast {
    success(title: string, msg: string) {

    }
}
