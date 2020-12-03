import { HttpClientTestingModule } from '@angular/common/http/testing';
import { Injector } from '@angular/core';
import { fakeAsync, getTestBed, TestBed, tick } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ApplicationService } from 'app/service/application/application.service';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import { MonitoringService, RouterService, UserService } from 'app/service/services.module';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { NgxsStoreModule } from 'app/store/store.module';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { Observable } from 'rxjs/Observable';
import { Pipeline } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { PipelineService } from '../../../../service/pipeline/pipeline.service';
import { SharedModule } from '../../../../shared/shared.module';
import { ToastService } from '../../../../shared/toast/ToastService';
import { PipelineModule } from '../../pipeline.module';
import { PipelineAdminComponent } from './pipeline.admin.component';

describe('CDS: Pipeline Admin Component', () => {

    let injector: Injector;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            providers: [
                PipelineService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                { provide: ToastService, useClass: MockToast },
                { provide: Router, useClass: MockRouter },
                NavbarService,
                TranslateService,
                TranslateLoader,
                WorkflowService,
                WorkflowRunService,
                TranslateParser,
                UserService,
                ApplicationService,
                EnvironmentService,
                AuthenticationService,
                MonitoringService,
                ProjectStore,
                RouterService,
                ProjectService
            ],
            imports: [
                PipelineModule,
                NgxsStoreModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ]
        }).compileComponents();

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });

    it('should update pipeline', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(PipelineAdminComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let p: Project = new Project();
        p.key = 'key1';
        p.permissions = {
            readable : true,
            writable: true,
            executable: true
        }
        fixture.componentInstance.project = p;

        let pip: Pipeline = new Pipeline();
        pip.name = 'pipName';
        fixture.componentInstance.pipeline = pip;


        fixture.detectChanges();
        tick(250);

        let store: Store = injector.get(Store);
        spyOn(store, 'dispatch').and.callFake(() => of(null));
        fixture.debugElement.nativeElement.querySelector('.ui.button.green.button').click();

        expect(store.dispatch).toHaveBeenCalledTimes(1);
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = Observable.of({ key: 'key1', pipName: 'pip1' });
        this.queryParams = Observable.of({ key: 'key1', appName: 'pip1', tab: 'workflow' });
    }
}

class MockRouter {
    public navigate() {
    }
}
