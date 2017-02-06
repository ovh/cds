/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, tick, getTestBed} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, ResponseOptions, Response} from '@angular/http';
import {ActionComponent} from './action.component';
import {SharedService} from '../shared.service';
import {SharedModule} from '../shared.module';
import {RequirementStore} from '../../service/worker/requirement/requirement.store';
import {ParameterService} from '../../service/parameter/parameter.service';
import {RequirementService} from '../../service/worker/requirement/requirement.service';
import {Action} from '../../model/action.model';
import {RequirementEvent} from '../requirements/requirement.event.model';
import {Requirement} from '../../model/requirement.model';
import {Parameter} from '../../model/parameter.model';
import {ParameterEvent} from '../parameter/parameter.event.model';
import {ActionEvent} from './action.event.model';
import {ActionStore} from '../../service/action/action.store';
import {ActionService} from '../../service/action/action.service';
import {Injector} from '@angular/core';
import {RepoManagerService} from '../../service/repomanager/project.repomanager.service';

describe('CDS: Action Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                TranslateService,
                RequirementStore,
                RequirementService,
                ParameterService,
                RepoManagerService,
                ActionStore,
                ActionService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });

        injector = getTestBed();
        backend = injector.get(XHRBackend);
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });



    it('should create and then delete a requirement', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        fixture.componentInstance.editableAction = action;

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

    it('should create and then delete a parameter', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        fixture.componentInstance.editableAction = action;

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

    it('should send delete action event', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        fixture.componentInstance.editableAction = action;

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;

        spyOn(fixture.componentInstance.actionEvent, 'emit');
        compiled.querySelector('.ui.red.button').click();
        fixture.detectChanges();
        tick(50);
        compiled.querySelector('.ui.red.button.active').click();

        expect(compiled.querySelector('button[name="updatebtn"]')).toBeFalsy();
        expect(fixture.componentInstance.actionEvent.emit).toHaveBeenCalledWith(new ActionEvent('delete', action));
    }));

    it('should send update action event', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        fixture.componentInstance.editableAction = action;

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;

        spyOn(fixture.componentInstance.actionEvent, 'emit');
        let inputName = compiled.querySelector('input[name="actionName"]');
        inputName.dispatchEvent(new Event('keydown'));

        fixture.detectChanges();
        tick(50);

        expect(compiled.querySelector('button[name="deletebtn"]')).toBeFalsy();

        let btn = compiled.querySelector('button[name="updatebtn"]');
        btn.click();

        expect(fixture.componentInstance.actionEvent.emit).toHaveBeenCalledWith(new ActionEvent('update', action));
    }));

    it('should add and then remove a step', fakeAsync( () => {
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '[{ "name" : "action1" }]'})));
        });

        // Create component
        let fixture = TestBed.createComponent(ActionComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        expect(backend.connectionsArray[0].request.url).toBe('/action', 'Component must load public action');

        let action: Action = new Action();
        action.name = 'FooAction';
        action.requirements = new Array<Requirement>();
        fixture.componentInstance.editableAction = action;

        fixture.detectChanges();
        tick(50);


        fixture.componentInstance.selectPublicAction('action1');
        expect(fixture.componentInstance.selectedStep.name).toBe('action1');

        let compiled = fixture.debugElement.nativeElement;

        spyOn(fixture.componentInstance.actionEvent, 'emit');
        let inputName = compiled.querySelector('button[name="addstepbtn"]');
        inputName.click();

        expect(fixture.componentInstance.editableAction.actions.length).toBe(1, 'Action must have 1 step');
        expect(fixture.componentInstance.editableAction.actions[0].name).toBe('action1');

        fixture.detectChanges();
        tick(50);

        compiled.querySelector('.ui.red.button').click();

        fixture.detectChanges();
        tick(50);

        compiled.querySelector('.ui.red.button.active').click();


        expect(fixture.componentInstance.editableAction.actions.length).toBe(0, 'Action must have 0 step');
    }));
});

