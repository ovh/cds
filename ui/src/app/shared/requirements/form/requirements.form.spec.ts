/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {RequirementService} from '../../../service/worker/requirement/requirement.service';
import {SharedModule} from '../../shared.module';
import {RequirementsFormComponent} from './requirements.form.component';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';
import {RequirementStore} from '../../../service/worker/requirement/requirement.store';

describe('CDS: Variable From Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                RequirementService,
                RequirementStore,
                TranslateService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

        injector = getTestBed();
        backend = injector.get(XHRBackend);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });


    it('should create a new requirement and auto write value', fakeAsync( () => {
        let call = 0;
        // Mock Http request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '["binary", "network"]'})));
        });


        // Create component
        let fixture = TestBed.createComponent(RequirementsFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        let r = new Requirement('binary');
        r.name = 'foo';
        r.value = 'foo';

        fixture.detectChanges();
        tick(50);

        // simulate typing new variable
        let inputName = compiled.querySelector('input[name="name"]');
        inputName.value = r.name;
        inputName.dispatchEvent(new Event('input'));
        inputName.dispatchEvent(new Event('keyup'));

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.event, 'emit');
        compiled.querySelector('.ui.blue.button').click();

        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(new RequirementEvent('add', r));
    }));

    it('should create a new requirement and auto write name', fakeAsync( () => {
        let call = 0;
        // Mock Http request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '["binary", "network"]'})));
        });


        // Create component
        let fixture = TestBed.createComponent(RequirementsFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        let r = new Requirement('binary');
        r.name = 'foo';
        r.value = 'foo';

        fixture.detectChanges();
        tick(50);

        // simulate typing new variable
        let inputName = compiled.querySelector('input[name="value"]');
        inputName.value = r.value;
        inputName.dispatchEvent(new Event('input'));
        inputName.dispatchEvent(new Event('keyup'));

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.event, 'emit');
        compiled.querySelector('.ui.blue.button').click();

        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(new RequirementEvent('add', r));
    }));
});

