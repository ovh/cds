import { APP_BASE_HREF } from '@angular/common';
import { HttpRequest, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { fakeAsync, flush, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Action } from 'app/model/action.model';
import { Parameter } from 'app/model/parameter.model';
import { Project } from 'app/model/project.model';
import { Requirement } from 'app/model/requirement.model';
import { ActionService } from 'app/service/action/action.service';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { ParameterService } from 'app/service/parameter/parameter.service';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { RequirementService } from 'app/service/requirement/requirement.service';
import { RequirementStore } from 'app/service/requirement/requirement.store';
import { UserService } from 'app/service/user/user.service';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { ParameterEvent } from '../parameter/parameter.event.model';
import { RequirementEvent } from '../requirements/requirement.event.model';
import { SharedModule } from '../shared.module';
import { SharedService } from '../shared.service';
import { ActionComponent } from './action.component';
import { ActionEvent } from './action.event.model';
import { StepEvent } from './step/step.event';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ActionAsCodeService, ConfigService, EntityService } from 'app/service/services.module';

describe('CDS: Action Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            providers: [
                EntityService,
                SharedService,
                TranslateService,
                RequirementStore,
                RequirementService,
                ParameterService,
                RepoManagerService,
                ActionService,
                WorkerModelService,
                TranslateLoader,
                TranslateParser,
                { provide: APP_BASE_HREF, useValue: '/' },
                UserService,
                AuthenticationService,
                ActionAsCodeService,
                ConfigService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting()
            ],
            imports: [
                BrowserAnimationsModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot()
            ]
        }).compileComponents();
    });

    it('should create and then delete a requirement', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        fixture.componentInstance.editableAction = action;
        fixture.componentInstance.edit = true;
        fixture.componentInstance.project = <Project>{ key: 'key' };

        let r: Requirement = new Requirement('binary');
        r.name = 'npm';
        r.value = 'npm';

        // Add a requirement
        let addRequirementEvent: RequirementEvent = new RequirementEvent('add', r);
        fixture.componentInstance.requirementEvent(addRequirementEvent);
        expect(fixture.componentInstance.editableAction.requirements.length).toBe(1, 'Action must have 1 requirement');
        expect(fixture.componentInstance.editableAction.requirements[0]).toBe(r);

        // Not add twice
        fixture.componentInstance.requirementEvent(addRequirementEvent);
        expect(fixture.componentInstance.editableAction.requirements.length).toBe(1, 'Action must have 1 requirement');
        expect(fixture.componentInstance.editableAction.requirements[0]).toBe(r);

        // Remove a requirement
        let removeRequiementEvent: RequirementEvent = new RequirementEvent('delete', r);
        fixture.componentInstance.requirementEvent(removeRequiementEvent);
        expect(fixture.componentInstance.editableAction.requirements.length).toBe(0, 'Action must have 0 requirement');
    }));

    it('should create and then delete a parameter', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        fixture.componentInstance.editableAction = action;
        fixture.componentInstance.edit = true;
        fixture.componentInstance.project = <Project>{ key: 'key' };

        let p: Parameter = new Parameter();
        p.name = 'gitUrl';
        p.type = 'string';
        p.description = 'git url of the repository';

        // Add a parameter
        let addparamEvent: ParameterEvent = new ParameterEvent('add', p);
        fixture.componentInstance.parameterEvent(addparamEvent);
        expect(fixture.componentInstance.editableAction.parameters.length).toBe(1, 'Action must have 1 parameter');
        expect(fixture.componentInstance.editableAction.parameters[0]).toBe(p);

        // Remove a parameter
        let removeParamEvent: ParameterEvent = new ParameterEvent('delete', p);
        fixture.componentInstance.parameterEvent(removeParamEvent);
        expect(fixture.componentInstance.editableAction.parameters.length).toBe(0, 'Action must have 0 parameter');
    }));

    it('should send insert action event', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        fixture.componentInstance.editableAction = action;
        fixture.componentInstance.edit = true;
        fixture.componentInstance.project = <Project>{ key: 'key' };

        fixture.detectChanges();
        tick(100);

        let compiled = fixture.debugElement.nativeElement;

        spyOn(fixture.componentInstance.actionEvent, 'emit');
        let inputName = compiled.querySelector('input[name="actionName"]');
        inputName.dispatchEvent(new Event('keydown'));

        fixture.detectChanges();
        tick(50);

        expect(compiled.querySelector('button[name="deletebtn"]')).toBeFalsy();
        expect(compiled.querySelector('button[name="updatebtn"]')).toBeFalsy();

        let btn = compiled.querySelector('button[name="updateBtn"]');
        btn.click();

        expect(fixture.componentInstance.actionEvent.emit).toHaveBeenCalledWith(new ActionEvent('update', action));

        flush();
    }));

    it('should add and then remove a step', fakeAsync(() => {
        const http = TestBed.get(HttpTestingController);

        let actionMock = <Action>{
            name: 'action1'
        };

        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        fixture.componentInstance.project = <Project>{ key: 'key' };

        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();


        fixture.componentInstance.ngOnInit();
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/key/action')).flush(actionMock);


        let action: Action = <Action>{
            name: 'FooAction',
            requirements: new Array<Requirement>()
        };
        fixture.componentInstance.editableAction = action;
        fixture.componentInstance.edit = true;

        let step = <Action>{
            always_executed: false,
            name: 'action1'
        };
        let event = new StepEvent('add', step);
        fixture.componentInstance.stepManagement(event);

        expect(fixture.componentInstance.steps.length).toBe(1, 'Action must have 1 step');
        expect(fixture.componentInstance.steps[0].name).toBe('action1');

        event.type = 'add';
        step.always_executed = true;
        step.name = 'action2';
        fixture.componentInstance.stepManagement(event);
        expect(fixture.componentInstance.steps.length).toBe(2, 'Action must have 2 steps');
        expect(fixture.componentInstance.steps[1].name).toBe('action2');

        flush();
    }));

    it('should init step not always executed and step always executed', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action = new Action();
        action.name = 'rootAction';

        let step1 = new Action();
        step1.always_executed = true;

        let step2 = new Action();
        step2.always_executed = false;

        action.actions = new Array<Action>();
        action.actions.push(step1, step2);

        fixture.componentInstance.action = action;

        expect(fixture.componentInstance.steps.length).toBe(2);
    }));
});
