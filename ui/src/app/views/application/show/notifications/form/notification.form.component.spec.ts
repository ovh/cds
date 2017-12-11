/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed, tick} from '@angular/core/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector, EventEmitter} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from '@ngx-translate/core';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {ApplicationService} from '../../../../../service/application/application.service';
import {ProjectStore} from '../../../../../service/project/project.store';
import {ProjectService} from '../../../../../service/project/project.service';
import {PipelineService} from '../../../../../service/pipeline/pipeline.service';
import {EnvironmentService} from '../../../../../service/environment/environment.service';
import {VariableService} from '../../../../../service/variable/variable.service';
import {AuthentificationStore} from '../../../../../service/auth/authentification.store';
import {SharedModule} from '../../../../../shared/shared.module';
import {ApplicationNotificationFormModalComponent} from './notification.form.component';
import {ApplicationModule} from '../../../application.module';
import {Project} from '../../../../../model/project.model';
import {Application, ApplicationPipeline} from '../../../../../model/application.model';
import {UserNotificationSettings, UserNotificationTemplate, Notification} from '../../../../../model/notification.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {Environment} from '../../../../../model/environment.model';
import {NotificationEvent} from '../notification.event';
import {Operator} from 'rxjs/Operator';
import {Observable} from 'rxjs/Observable';
import {Subscriber} from 'rxjs/Subscriber';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Application notifications', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: XHRBackend, useClass: MockBackend},
                AuthentificationStore,
                ApplicationStore,
                ApplicationService,
                ProjectStore,
                ProjectService,
                PipelineService,
                EnvironmentService,
                VariableService,
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports: [
                ApplicationModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });

    it('should create new multiple notifications', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationNotificationFormModalComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init component Input
        let p = new Project();
        p.environments = new Array<Environment>();
        let e = new Environment();
        e.name = 'production';
        p.environments.push(e);
        fixture.componentInstance.project = p;

        let a = new Application();
        a.pipelines = new Array<ApplicationPipeline>();
        let ap1 = new ApplicationPipeline();
        ap1.pipeline = new Pipeline();
        ap1.pipeline.name = 'build1';
        let ap2 = new ApplicationPipeline();
        ap2.pipeline = new Pipeline();
        ap2.pipeline.type = 'deployment';
        ap2.pipeline.name = 'deploy1';
        a.pipelines.push(ap1, ap2);
        fixture.componentInstance.application = a;

        // Notification to add
        fixture.componentInstance.initForm();
        let userNotif = new UserNotificationSettings();
        userNotif.on_failure = 'change';
        userNotif.on_success = 'never';
        userNotif.template = new UserNotificationTemplate();
        userNotif.template.subject = 'My Subject';
        userNotif.template.body = 'My Body';
        fixture.componentInstance.selected = {
            pipeline: [ap1.pipeline.name, ap2.pipeline.name],
            environment: [e.name],
            notification: userNotif,
            type: 'jabber',
            recipients: 'toto'
        };
        fixture.componentInstance.onAuthorControl.setValue(true);
        fixture.componentInstance.onStartControl.setValue(true);

        fixture.componentInstance.event = new MockEventEmitter();

        fixture.detectChanges();
        tick(50);

        fixture.componentInstance.sendEvent();

        let result: MockEventEmitter = <MockEventEmitter>fixture.componentInstance.event;
        expect(result.event.type).toBe('add');
        expect(result.event.notifications.length).toBe(2);
        expect(result.event.notifications[0].pipeline.name).toBe(ap1.pipeline.name);
        expect(result.event.notifications[0].environment).toBeFalsy();
        expect(result.event.notifications[0].notifications['jabber']).toBeTruthy();
        expect(result.event.notifications[0].notifications['jabber'].on_failure).toBe('change');
        expect(result.event.notifications[0].notifications['jabber'].on_success).toBe('never');
        expect(result.event.notifications[0].notifications['jabber'].on_start).toBeTruthy();
        expect(result.event.notifications[0].notifications['jabber'].template.subject).toBe('My Subject');
        expect(result.event.notifications[0].notifications['jabber'].template.body).toBe('My Body');
        expect(result.event.notifications[0].notifications['jabber'].send_to_author).toBeTruthy();
        expect(result.event.notifications[0].notifications['jabber'].send_to_groups).toBeFalsy();

        expect(result.event.notifications[1].pipeline.name).toBe(ap2.pipeline.name);
        expect(result.event.notifications[1].environment.name).toBe(e.name);
        expect(result.event.notifications[1].notifications['jabber']).toBeTruthy();
        expect(result.event.notifications[1].notifications['jabber'].on_failure).toBe('change');
        expect(result.event.notifications[1].notifications['jabber'].on_success).toBe('never');
        expect(result.event.notifications[1].notifications['jabber'].on_start).toBeTruthy();
        expect(result.event.notifications[1].notifications['jabber'].template.subject).toBe('My Subject');
        expect(result.event.notifications[1].notifications['jabber'].template.body).toBe('My Body');
        expect(result.event.notifications[1].notifications['jabber'].send_to_author).toBeTruthy();
        expect(result.event.notifications[1].notifications['jabber'].send_to_groups).toBeFalsy();
    }));


    it('should add one type of notif to an existing notif', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationNotificationFormModalComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init component Input
        let p = new Project();
        p.environments = new Array<Environment>();
        let e = new Environment();
        e.name = 'production';
        p.environments.push(e);
        fixture.componentInstance.project = p;

        let a = new Application();
        a.pipelines = new Array<ApplicationPipeline>();
        let ap1 = new ApplicationPipeline();
        ap1.id = 1;
        ap1.pipeline = new Pipeline();
        ap1.pipeline.name = 'build1';
        let ap2 = new ApplicationPipeline();
        ap2.id = 2;
        ap2.pipeline = new Pipeline();
        ap2.pipeline.type = 'deployment';
        ap2.pipeline.name = 'deploy1';
        a.pipelines.push(ap1, ap2);

        a.notifications = new Array<Notification>();
        let n = new Notification();
        n.pipeline = ap2.pipeline;
        n.environment = e;
        n.application_pipeline_id = ap2.id;
        n.notifications['jabber'] = new UserNotificationSettings();
        (<UserNotificationSettings>n.notifications['jabber']).on_failure = 'never';
        (<UserNotificationSettings>n.notifications['jabber']).on_start = true;
        (<UserNotificationSettings>n.notifications['jabber']).on_success = 'never';
        (<UserNotificationSettings>n.notifications['jabber']).recipients = ['foo', 'bar'];
        (<UserNotificationSettings>n.notifications['jabber']).send_to_author = true;
        (<UserNotificationSettings>n.notifications['jabber']).send_to_groups = false;
        a.notifications.push(n);

        fixture.componentInstance.application = a;

        // Notification to add
        fixture.componentInstance.initForm();
        let userNotif = new UserNotificationSettings();
        userNotif.on_failure = 'change';
        userNotif.on_success = 'never';
        userNotif.template = new UserNotificationTemplate();
        userNotif.template.subject = 'My Subject';
        userNotif.template.body = 'My Body';
        fixture.componentInstance.selected = {
            pipeline: [ap2.pipeline.name],
            environment: [e.name],
            notification: userNotif,
            type: 'email',
            recipients: 'toto'
        };
        fixture.componentInstance.onAuthorControl.setValue(true);
        fixture.componentInstance.onStartControl.setValue(true);

        fixture.componentInstance.event = new MockEventEmitter();
        fixture.componentInstance.sendEvent();

        let result: MockEventEmitter = <MockEventEmitter>fixture.componentInstance.event;
        expect(result.event.type).toBe('add');
        expect(result.event.notifications.length).toBe(1);
        expect(result.event.notifications[0].pipeline.name).toBe(ap2.pipeline.name);
        expect(result.event.notifications[0].environment.name).toBe(e.name);
        expect(Object.keys(result.event.notifications[0].notifications).length).toBe(2);
        expect(result.event.notifications[0].notifications['email']).toBeTruthy();
        expect(result.event.notifications[0].notifications['email'].on_failure).toBe('change');
        expect(result.event.notifications[0].notifications['email'].on_success).toBe('never');
        expect(result.event.notifications[0].notifications['email'].on_start).toBeTruthy();
        expect(result.event.notifications[0].notifications['email'].template.subject).toBe('My Subject');
        expect(result.event.notifications[0].notifications['email'].template.body).toBe('My Body');
        expect(result.event.notifications[0].notifications['email'].send_to_author).toBeTruthy();
        expect(result.event.notifications[0].notifications['email'].send_to_groups).toBeFalsy();
    }));

    it('should update a notification', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationNotificationFormModalComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Notification to add
        fixture.componentInstance.initForm();

        // Init component Input
        let p = new Project();
        p.environments = new Array<Environment>();
        let e = new Environment();
        e.name = 'production';
        p.environments.push(e);
        fixture.componentInstance.project = p;

        let a = new Application();
        a.pipelines = new Array<ApplicationPipeline>();
        let ap1 = new ApplicationPipeline();
        ap1.id = 1;
        ap1.pipeline = new Pipeline();
        ap1.pipeline.name = 'build1';
        let ap2 = new ApplicationPipeline();
        ap2.id = 2;
        ap2.pipeline = new Pipeline();
        ap2.pipeline.type = 'deployment';
        ap2.pipeline.name = 'deploy1';
        a.pipelines.push(ap1, ap2);

        a.notifications = new Array<Notification>();
        let n = new Notification();
        n.pipeline = ap2.pipeline;
        n.environment = e;
        n.application_pipeline_id = ap2.id;
        n.notifications['jabber'] = new UserNotificationSettings();
        (<UserNotificationSettings>n.notifications['jabber']).on_failure = 'never';
        (<UserNotificationSettings>n.notifications['jabber']).on_start = true;
        (<UserNotificationSettings>n.notifications['jabber']).on_success = 'never';
        (<UserNotificationSettings>n.notifications['jabber']).recipients = ['foo', 'bar'];
        (<UserNotificationSettings>n.notifications['jabber']).send_to_author = true;
        (<UserNotificationSettings>n.notifications['jabber']).send_to_groups = false;
        a.notifications.push(n);

        fixture.componentInstance.application = a;
        fixture.componentInstance.notification = n;

        fixture.detectChanges();
        tick(50);

        expect(fixture.componentInstance.selected.recipients).toBe('foo,bar');

        // Update notification
        fixture.componentInstance.onStartControl.setValue(false);
        fixture.componentInstance.selected.notification.template.subject = 'New Subject';

        // Run update
        fixture.componentInstance.event = new MockEventEmitter();
        fixture.componentInstance.sendEvent();

        let result: MockEventEmitter = <MockEventEmitter>fixture.componentInstance.event;
        expect(result.event.type).toBe('update');
        expect(result.event.notifications.length).toBe(1);
        expect(result.event.notifications[0].notifications['jabber'].on_start).toBeFalsy();
        expect(result.event.notifications[0].notifications['jabber'].template.subject).toBe('New Subject');
    }));

    it('should remove on type of notif', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationNotificationFormModalComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Notification to add
        fixture.componentInstance.initForm();

        // Init component Input
        let p = new Project();
        p.environments = new Array<Environment>();
        let e = new Environment();
        e.name = 'production';
        p.environments.push(e);
        fixture.componentInstance.project = p;

        let a = new Application();
        a.pipelines = new Array<ApplicationPipeline>();
        let ap1 = new ApplicationPipeline();
        ap1.id = 1;
        ap1.pipeline = new Pipeline();
        ap1.pipeline.name = 'build1';
        let ap2 = new ApplicationPipeline();
        ap2.id = 2;
        ap2.pipeline = new Pipeline();
        ap2.pipeline.type = 'deployment';
        ap2.pipeline.name = 'deploy1';
        a.pipelines.push(ap1, ap2);

        a.notifications = new Array<Notification>();
        let n = new Notification();
        n.pipeline = ap2.pipeline;
        n.environment = e;
        n.application_pipeline_id = ap2.id;

        let jabber = new UserNotificationSettings();
        jabber.on_failure = 'never';
        jabber.on_start = true;
        jabber.on_success = 'never';
        jabber.recipients = ['foo', 'bar'];
        jabber.send_to_author = true;
        jabber.send_to_groups = false;
        n.notifications['jabber'] = jabber;

        let email = new UserNotificationSettings();
        email.on_failure = 'never';
        email.on_start = true;
        email.on_success = 'never';
        email.recipients = ['foo', 'bar'];
        email.send_to_author = true;
        email.send_to_groups = false;
        n.notifications['email'] = email;

        a.notifications.push(n);

        fixture.componentInstance.application = a;

        let notifToDelete = new Notification();
        notifToDelete.application_pipeline_id = n.application_pipeline_id;
        notifToDelete.pipeline = n.pipeline;
        notifToDelete.environment = n.environment;
        notifToDelete.notifications['email'] = email;
        fixture.componentInstance.notification = notifToDelete;

        fixture.detectChanges();
        tick(50);

        // Run update
        fixture.componentInstance.event = new MockEventEmitter();
        fixture.componentInstance.removeNotif();

        let result: MockEventEmitter = <MockEventEmitter>fixture.componentInstance.event;
        expect(result.event.type).toBe('update');
        expect(result.event.notifications.length).toBe(1);
        expect(result.event.notifications[0].notifications['jabber']).toBeTruthy();
        expect(result.event.notifications[0].notifications['email']).toBeFalsy();
    }));
    it('should remove last type of notif', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationNotificationFormModalComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Notification to add
        fixture.componentInstance.initForm();

        // Init component Input
        let p = new Project();
        p.environments = new Array<Environment>();
        let e = new Environment();
        e.name = 'production';
        p.environments.push(e);
        fixture.componentInstance.project = p;

        let a = new Application();
        a.pipelines = new Array<ApplicationPipeline>();
        let ap1 = new ApplicationPipeline();
        ap1.id = 1;
        ap1.pipeline = new Pipeline();
        ap1.pipeline.name = 'build1';
        let ap2 = new ApplicationPipeline();
        ap2.id = 2;
        ap2.pipeline = new Pipeline();
        ap2.pipeline.type = 'deployment';
        ap2.pipeline.name = 'deploy1';
        a.pipelines.push(ap1, ap2);

        a.notifications = new Array<Notification>();
        let n = new Notification();
        n.pipeline = ap2.pipeline;
        n.environment = e;
        n.application_pipeline_id = ap2.id;

        let jabber = new UserNotificationSettings();
        jabber.on_failure = 'never';
        jabber.on_start = true;
        jabber.on_success = 'never';
        jabber.recipients = ['foo', 'bar'];
        jabber.send_to_author = true;
        jabber.send_to_groups = false;
        n.notifications['jabber'] = jabber;

        a.notifications.push(n);

        fixture.componentInstance.application = a;

        let notifToDelete = new Notification();
        notifToDelete.application_pipeline_id = n.application_pipeline_id;
        notifToDelete.pipeline = n.pipeline;
        notifToDelete.environment = n.environment;
        notifToDelete.notifications['jabber'] = jabber;
        fixture.componentInstance.notification = notifToDelete;

        fixture.detectChanges();
        tick(50);

        // Run update
        fixture.componentInstance.event = new MockEventEmitter();
        fixture.componentInstance.removeNotif();

        let result: MockEventEmitter = <MockEventEmitter>fixture.componentInstance.event;
        expect(result.event.type).toBe('delete');
        expect(Object.keys(result.event.notifications[0].notifications).length).toBe(0);
    }));


    it('should clone notification', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ApplicationNotificationFormModalComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Notification to add
        fixture.componentInstance.initForm();

        // Init component Input
        let p = new Project();
        p.environments = new Array<Environment>();
        let e = new Environment();
        e.name = 'production';
        p.environments.push(e);
        p.applications = new Array<Application>();

        let a = new Application();
        a.name = 'app';
        a.pipelines = new Array<ApplicationPipeline>();
        let ap1 = new ApplicationPipeline();
        ap1.id = 1;
        ap1.pipeline = new Pipeline();
        ap1.pipeline.name = 'build1';
        let ap2 = new ApplicationPipeline();
        ap2.id = 2;
        ap2.pipeline = new Pipeline();
        ap2.pipeline.type = 'deployment';
        ap2.pipeline.name = 'deploy1';
        a.pipelines.push(ap1, ap2);

        a.notifications = new Array<Notification>();
        let n = new Notification();
        n.pipeline = ap2.pipeline;
        n.environment = e;
        n.application_pipeline_id = ap2.id;

        let jabber = new UserNotificationSettings();
        jabber.on_failure = 'never';
        jabber.on_start = true;
        jabber.on_success = 'never';
        jabber.recipients = ['foo', 'bar'];
        jabber.send_to_author = true;
        jabber.send_to_groups = false;
        jabber.template.subject = 'subj';
        jabber.template.body = 'body';
        n.notifications['jabber'] = jabber;

        a.notifications.push(n);

        fixture.componentInstance.application = a;

        p.applications.push(a);
        fixture.componentInstance.project = p;

        fixture.componentInstance.projectNotifications = new Array<Notification>();
        fixture.componentInstance.projectNotifications.push(n);

        expect(fixture.componentInstance.getCloneNotificationLabel(n)).toBe('[jabber] app-deploy1-production');

        fixture.componentInstance.updateWithClonedNotification(0);

        expect(fixture.componentInstance.isNewNotif).toBeTruthy();
        expect(fixture.componentInstance.selected.clonedType).toBe('jabber');
        expect(fixture.componentInstance.selected.notification.on_failure).toBe('never');
        expect(fixture.componentInstance.selected.notification.on_success).toBe('never');
        expect(fixture.componentInstance.selected.notification.template.subject).toBe('subj');
        expect(fixture.componentInstance.selected.notification.template.body).toBe('body');
        expect(fixture.componentInstance.onStartControl.value).toBeTruthy();
        expect(fixture.componentInstance.onAuthorControl.value).toBeTruthy();
        expect(fixture.componentInstance.onGroupsControl.value).toBeFalsy();
    }));
});

class MockEventEmitter extends EventEmitter<NotificationEvent> {
    event: NotificationEvent;
    __isAsync: any;



    constructor() {
        super();
    }

    emit(ne: NotificationEvent): void {
        this.event = ne;
    }
}
