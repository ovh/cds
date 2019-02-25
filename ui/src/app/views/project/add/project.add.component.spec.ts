/* tslint:disable:no-unused-variable */
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { CUSTOM_ELEMENTS_SCHEMA, Injector } from '@angular/core';
import { getTestBed, TestBed } from '@angular/core/testing';
import { XHRBackend } from '@angular/http';
import { MockBackend } from '@angular/http/testing';
import { Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ToasterService } from 'angular2-toaster/angular2-toaster';
import { AddProject } from 'app/store/project.action';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { Group, GroupPermission } from '../../../model/group.model';
import { Project } from '../../../model/project.model';
import { EnvironmentService } from '../../../service/environment/environment.service';
import { GroupService } from '../../../service/group/group.service';
import { NavbarService } from '../../../service/navbar/navbar.service';
import { PipelineService } from '../../../service/pipeline/pipeline.service';
import { ProjectService } from '../../../service/project/project.service';
import { ProjectStore } from '../../../service/project/project.store';
import { RepoManagerService } from '../../../service/repomanager/project.repomanager.service';
import { VariableService } from '../../../service/variable/variable.service';
import { SharedModule } from '../../../shared/shared.module';
import { ToastService } from '../../../shared/toast/ToastService';
import { ProjectModule } from '../project.module';
import { ProjectAddComponent } from './project.add.component';
describe('CDS: Project Show Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                RepoManagerService,
                ProjectStore,
                NavbarService,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService,
                ToasterService,
                TranslateService,
                TranslateParser,
                GroupService,
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
        });
        injector = getTestBed();
        backend = injector.get(MockBackend);

    });

    afterEach(() => {
        injector = undefined;
    });


    it('it should create a project', () => {
        let store: Store = injector.get(Store);
        let router: Router = injector.get(Router);

        spyOn(store, 'dispatch').and.callFake(() => {
            return of(null);
        });

        spyOn(router, 'navigate').and.callFake(() => {
            return;
        });

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
