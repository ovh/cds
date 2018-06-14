/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import {VariableService} from '../../../service/variable/variable.service';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {SharedService} from '../../shared.service';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {VariableFormComponent} from './variable.form';
import {GroupService} from '../../../service/group/group.service';
import {Variable} from '../../../model/variable.model';
import {VariableEvent} from '../variable.event.model';
import {SharedModule} from '../../shared.module';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Variable From Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                VariableService,
                GroupService,
                SharedService,
                TranslateService,
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                SharedModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
        backend = injector.get(MockBackend);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });


    it('Create new variable', fakeAsync( () => {
        let call = 0;
        // Mock Http request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '["string", "password"]'})));
        });


        // Create component
        let fixture = TestBed.createComponent(VariableFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(50);

        expect(fixture.debugElement.nativeElement.querySelector('.ui.button.disabled')).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        let variable = new Variable();
        variable.name = 'foo';
        variable.type = 'string';
        variable.value = 'bar';

        fixture.detectChanges();
        tick(50);

        // simulate typing new variable
        let inputName = compiled.querySelector('input[name="name"]');
        inputName.value = variable.name;
        inputName.dispatchEvent(new Event('input'));

        fixture.componentInstance.newVariable.type = variable.type;

        fixture.detectChanges();
        tick(50);

        let inputValue = compiled.querySelector('input[name="value"]');
        inputValue.value = variable.value;
        inputValue.dispatchEvent(new Event('input'));
        inputValue.dispatchEvent(new Event('change'));

        spyOn(fixture.componentInstance.createVariableEvent, 'emit');
        compiled.querySelector('.ui.green.button').click();

        expect(fixture.componentInstance.createVariableEvent.emit).toHaveBeenCalledWith(new VariableEvent('add', variable));
    }));
});

