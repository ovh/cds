/* eslint-disable @typescript-eslint/no-unused-vars */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';
import {APP_BASE_HREF} from '@angular/common';
import {VariableService} from '../../../service/variable/variable.service';
import {SharedService} from '../../shared.service';
import {Variable} from '../../../model/variable.model';
import {SharedModule} from '../../shared.module';
import {VariableEvent} from '../variable.event.model';
import {ProjectAuditService} from '../../../service/project/project.audit.service';
import {EnvironmentAuditService} from '../../../service/environment/environment.audit.service';
import {ApplicationAuditService} from '../../../service/application/application.audit.service';
import {VariableComponent} from './variable.component';

describe('CDS: Variable List Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                VariableService,
                SharedService,
                TranslateService,
                TranslateLoader,
                TranslateParser,
                ProjectAuditService,
                EnvironmentAuditService,
                ApplicationAuditService,
                { provide: APP_BASE_HREF, useValue : '/' }
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                SharedModule,
                TranslateModule.forRoot(),
                HttpClientTestingModule
            ]
        }).compileComponents();
    });


    it('Load Component + update value', fakeAsync(  () => {
        const http = TestBed.get(HttpTestingController);

        let mock = ['string', 'password'];

        // Create component
        let fixture = TestBed.createComponent(VariableComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        http.expectOne(((req: HttpRequest<any>) => req.url === '/variable/type')).flush(mock);

        let vars: Variable[] = [];
        let variable: Variable = new Variable();
        variable.name = 'foo';
        variable.type = 'string';
        variable.description = 'foo is my variable';
        variable.value = 'bar';

        vars.push(variable);
        fixture.componentInstance.variables = vars;
        fixture.componentInstance._cd.detectChanges();
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
    }));
});

