import { HttpClient, HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ToasterModule } from 'angular2-toaster-sgu';
import { EventService } from 'app/event.service';
import { MonitoringStatus } from 'app/model/monitoring.model';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { HelpService } from 'app/service/help/help.service';
import { MonitoringService } from 'app/service/monitoring/monitoring.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { of } from 'rxjs';
import 'rxjs/add/observable/of';
import { AppComponent } from './app.component';
import { createTranslateLoader } from './app.module';
import { AppService } from './app.service';
import { Application } from './model/application.model';
import { Pipeline } from './model/pipeline.model';
import { Project } from './model/project.model';
import { AuthSummary } from './model/user.model';
import { ApplicationService } from './service/application/application.service';
import { AuthenticationService } from './service/authentication/authentication.service';
import { BroadcastService } from './service/broadcast/broadcast.service';
import { BroadcastStore } from './service/broadcast/broadcast.store';
import { NavbarService } from './service/navbar/navbar.service';
import { NotificationService } from './service/notification/notification.service';
import { PipelineService } from './service/pipeline/pipeline.service';
import { ProjectService } from './service/project/project.service';
import { ProjectStore } from './service/project/project.store';
import { RouterService } from './service/router/router.service';
import { ThemeStore } from './service/theme/theme.store';
import { TimelineService } from './service/timeline/timeline.service';
import { TimelineStore } from './service/timeline/timeline.store';
import { UserService } from './service/user/user.service';
import { SharedModule } from './shared/shared.module';
import { ToastService } from './shared/toast/ToastService';
import { FetchCurrentAuth } from './store/authentication.action';
import { NgxsStoreModule } from './store/store.module';
import { NavbarModule } from './views/navbar/navbar.module';

describe('App: CDS', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                AppComponent
            ],
            providers: [
                Store,
                WorkflowService,
                WorkflowRunService,
                UserService,
                NavbarService,
                HelpService,
                MonitoringService,
                EventService,
                ProjectStore,
                EnvironmentService,
                AuthenticationService,
                ThemeStore,
                NotificationService,
                AppService,
                RouterService,
                ToastService,
                BroadcastStore,
                BroadcastService,
                TimelineStore,
                TimelineService,
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                { provide: ProjectService, useClass: MockProjectService },
                { provide: ApplicationService, useClass: MockApplicationService },
                { provide: PipelineService, useClass: MockPipelineService },
                { provide: ActivatedRoute, useClass: MockActivatedRoutes }
            ],
            imports: [
                NgxsStoreModule,
                SharedModule,
                NavbarModule,
                ToasterModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule,
                TranslateModule.forRoot({
                    loader: {
                        provide: TranslateLoader,
                        useFactory: createTranslateLoader,
                        deps: [HttpClient]
                    }
                }),
            ]
        }).compileComponents();
    });

    it('should create the app', () => {
        let fixture = TestBed.createComponent(AppComponent);
        let app = fixture.debugElement.componentInstance;
        expect(app).toBeTruthy();
    });


    it('should render a navbar', () => {
        let fixture = TestBed.createComponent(AppComponent);
        let compiled = fixture.debugElement.nativeElement;

        expect(fixture.componentInstance.isConnected).toBeFalsy('IsConnected flag must be false');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must not have the css class "connected"');

        fixture.componentInstance.ngOnInit();

        const http = TestBed.get(HttpTestingController);

        http.expectOne((req: HttpRequest<any>) => req.url === '/mon/status').flush(<MonitoringStatus>{});

        const store = TestBed.get(Store);
        store.dispatch(new FetchCurrentAuth());


        http.expectOne(((req: HttpRequest<any>) => req.url === '/auth/me')).flush(<AuthSummary>{
            user: { username: 'someone' }
        });

        expect(fixture.componentInstance.isConnected).toBeTruthy('IsConnected flag must be true');
        expect(compiled.querySelector('#navbar.connected')).toBeFalsy('Nav bar must have connected css class');
    });
});

class MockActivatedRoutes {

    snapshot: {
        params: {}
    };

    constructor() {
        this.snapshot = {
            params: {
                key: 'key1',
                appName: 'app2',
                pipName: 'pip2'
            }
        };
    }
}

class MockProjectService extends ProjectService {
    callKEY1 = 0;
    getProject(key: string) {
        switch (key) {
            case 'key1':
                if (this.callKEY1 === 0) {
                    this.callKEY1++;
                    let proj = new Project();
                    proj.key = 'key1';
                    proj.name = 'project1';
                    proj.last_modified = '2017-05-11T10:20:22.874779+02:00';
                    return of(proj);
                } else {
                    let proj = new Project();
                    proj.key = 'key1';
                    proj.name = 'project1';
                    proj.last_modified = '2017-06-11T10:20:22.874779+02:00';
                    return of(proj);
                }
            case 'key2':
                let proj2 = new Project();
                proj2.key = 'key2';
                proj2.name = 'project2';
                proj2.last_modified = '2017-05-11T10:20:22.874779+02:00';
                return of(proj2);
        }

    }
}

class MockApplicationService extends ApplicationService {
    callAPP2 = 0;

    getApplication(key: string, appName: string, filter?: { branch: string, remote: string }) {
        if (key === 'key1') {
            if (appName === 'app1') {
                let app = new Application();
                app.name = 'app1';
                app.last_modified = '2017-05-11T10:20:22.874779+02:00';
                return of(app);
            }
            if (appName === 'app2') {
                if (this.callAPP2 === 0) {
                    this.callAPP2++;
                    let app = new Application();
                    app.name = 'app2';
                    app.last_modified = '2017-05-11T10:20:22.874779+02:00';
                    return of(app);
                } else {
                    let app = new Application();
                    app.name = 'app2';
                    app.last_modified = '2017-06-11T10:20:22.874779+02:00';
                    return of(app);
                }

            }
        }
        if (key === 'key2') {
            if (appName === 'app3') {
                let app = new Application();
                app.name = 'app3';
                app.last_modified = '2017-05-11T10:20:22.874779+02:00';
                return of(app);
            }
        }
    }
}

class MockPipelineService extends PipelineService {
    callPIP2 = 0;
    getPipeline(key: string, pipName: string) {
        if (key === 'key1') {
            if (pipName === 'pip1') {
                let pip = new Pipeline();
                pip.name = 'pip1';
                pip.last_modified = 1494490822;
                return of(pip);
            }
            if (pipName === 'pip2') {
                if (this.callPIP2 === 0) {
                    this.callPIP2++;
                    let pip = new Pipeline();
                    pip.name = 'pip1';
                    pip.last_modified = 1494490822;
                    return of(pip);
                } else {
                    let pip = new Pipeline();
                    pip.name = 'pip1';
                    pip.last_modified = 1497169222;
                    return of(pip);
                }

            }
        }
        if (key === 'key2') {
            if (pipName === 'pip3') {
                let pip = new Pipeline();
                pip.name = 'pip3';
                pip.last_modified = 1494490822;
                return of(pip);
            }
        }
    }
}
