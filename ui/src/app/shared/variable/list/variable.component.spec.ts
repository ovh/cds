/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync, inject} from '@angular/core/testing';
import {VariableComponent} from './variable.component';
import {VariableService} from '../../../service/variable/variable.service';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {SharedService} from '../../shared.service';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Variable} from '../../../model/variable.model';
import {Injector} from '@angular/core';
import {SharedModule} from '../../shared.module';
import {VariableEvent} from '../variable.event.model';
import {ProjectAuditService} from '../../../service/project/project.audit.service';
import {EnvironmentAuditService} from '../../../service/environment/environment.audit.service';
import {ApplicationAuditService} from '../../../service/application/application.audit.service';

describe('CDS: Variable List Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                VariableService,
                SharedService,
                TranslateService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser,
                ProjectAuditService,
                EnvironmentAuditService,
                ApplicationAuditService
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });
    });


    it('Load Component + update value', fakeAsync(  inject([XHRBackend], (backend: MockBackend) => {
        // Mock Http request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '["string", "password"]'})));
        });

        // Create component
        let fixture = TestBed.createComponent(VariableComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        expect(backend.connectionsArray[0].request.url).toBe('/variable/type', 'Component must load variable type');

        let vars: Variable[] = [];
        let variable: Variable = new Variable();
        variable.name = 'foo';
        variable.type = 'string';
        variable.description = 'foo is my variable';
        variable.value = 'bar';

        vars.push(variable);
        fixture.componentInstance.variables = vars;

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.ui.red.button')).toBeTruthy('Delete button must be displayed');
        compiled.querySelector('.ui.red.button').click();

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.event, 'emit');

        expect(compiled.querySelector('.ui.buttons')).toBeTruthy('Confirmation buttons must be displayed');
        compiled.querySelector('.ui.red.button.active').click();

        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(
            new VariableEvent('delete', fixture.componentInstance.variables[0])
        );


        expect(fixture.componentInstance.variables[0].hasChanged).toBeFalsy('No update yet on this variable');
        expect(compiled.querySelector('button[name="btnupdatevar"]')).toBeFalsy('No Update, no button');

        let inputName = compiled.querySelector('input[name="varname"]');
        inputName.value = 'fooUpdated';
        inputName.dispatchEvent(new Event('keydown'));

        fixture.detectChanges();
        tick(100);

        expect(fixture.componentInstance.variables[0].hasChanged).toBeTruthy('No update yet on this variable');
        expect(compiled.querySelector('button[name="btnupdatevar"]')).toBeTruthy('No Update, no button');
        compiled.querySelector('button[name="btnupdatevar"]').click();
        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(
            new VariableEvent('update', fixture.componentInstance.variables[0])
        );
    })));
});

