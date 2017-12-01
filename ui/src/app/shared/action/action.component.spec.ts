/* tslint:disable:no-unused-variable */

import {fakeAsync, TestBed, tick} from '@angular/core/testing';
import {TranslateLoader, TranslateParser, TranslateService} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {ActionComponent} from './action.component';
import {SharedService} from '../shared.service';
import {SharedModule} from '../shared.module';
import {RequirementStore} from '../../service/worker-model/requirement/requirement.store';
import {ParameterService} from '../../service/parameter/parameter.service';
import {RequirementService} from '../../service/worker-model/requirement/requirement.service';
import {Action} from '../../model/action.model';
import {RequirementEvent} from '../requirements/requirement.event.model';
import {Requirement} from '../../model/requirement.model';
import {Parameter} from '../../model/parameter.model';
import {ParameterEvent} from '../parameter/parameter.event.model';
import {ActionEvent} from './action.event.model';
import {ActionStore} from '../../service/action/action.store';
import {ActionService} from '../../service/action/action.service';
import {RepoManagerService} from '../../service/repomanager/project.repomanager.service';
import {StepEvent} from './step/step.event';
import {WorkerModelService} from '../../service/worker-model/worker-model.service';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';

describe('CDS: Action Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                SharedService,
                TranslateService,
                RequirementStore,
                RequirementService,
                ParameterService,
                RepoManagerService,
                ActionStore,
                ActionService,
                WorkerModelService,
                {provide: XHRBackend, useClass: MockBackend},
                TranslateLoader,
                TranslateParser
            ],
            imports: [
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });
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

        fixture.detectChanges();
        tick(50);

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

        fixture.detectChanges();
        tick(50);

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

    it('should send delete action event', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        action.id = 1;
        fixture.componentInstance.editableAction = action;

        fixture.detectChanges();
        tick(50);

        // readonly , no button
        expect(fixture.debugElement.nativeElement.querySelector('.ui.red.button')).toBeFalsy();
        expect(fixture.debugElement.nativeElement.querySelector('button[name="updatebtn"]')).toBeFalsy();

        fixture.componentInstance.edit = true;

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;

        spyOn(fixture.componentInstance.actionEvent, 'emit');
        compiled.querySelector('.ui.red.button').click();
        fixture.detectChanges();
        tick(50);
        compiled.querySelector('.ui.red.button.active').click();

        expect(compiled.querySelector('button[name="updatebtn"]')).toBeTruthy();
        expect(fixture.componentInstance.actionEvent.emit).toHaveBeenCalledWith(new ActionEvent('delete', action));
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

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;

        spyOn(fixture.componentInstance.actionEvent, 'emit');
        let inputName = compiled.querySelector('input[name="actionName"]');
        inputName.dispatchEvent(new Event('keydown'));

        fixture.detectChanges();
        tick(50);

        expect(compiled.querySelector('button[name="deletebtn"]')).toBeFalsy();
        expect(compiled.querySelector('button[name="updatebtn"]')).toBeFalsy();

        let btn = compiled.querySelector('button[name="addbtn"]');
        btn.click();

        expect(fixture.componentInstance.actionEvent.emit).toHaveBeenCalledWith(new ActionEvent('insert', action));
    }));

    it('should add and then remove a step', fakeAsync(() => {
            const http = TestBed.get(HttpTestingController);

            let actionMock = new Action();
            actionMock.name = 'action1';


            // Create component
            let fixture = TestBed.createComponent(ActionComponent);
            let component = fixture.debugElement.componentInstance;
            expect(component).toBeTruthy();

            http.expectOne(((req: HttpRequest<any>) => {
                return req.url === '/action';
            })).flush(actionMock);

            let action: Action = new Action();
            action.name = 'FooAction';
            action.requirements = new Array<Requirement>();
            fixture.componentInstance.editableAction = action;
            fixture.componentInstance.edit = true;

            fixture.detectChanges();
            tick(50);


            let step = new Action();
            step.always_executed = false;
            step.name = 'action1';
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
        })
    );

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
})
;
