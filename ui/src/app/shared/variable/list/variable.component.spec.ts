/* tslint:disable:no-unused-variable */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {VariableComponent} from './variable.component';
import {VariableService} from '../../../service/variable/variable.service';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {SharedService} from '../../shared.service';
import {RouterTestingModule} from '@angular/router/testing';
import {Variable} from '../../../model/variable.model';
import {SharedModule} from '../../shared.module';
import {VariableEvent} from '../variable.event.model';
import {ProjectAuditService} from '../../../service/project/project.audit.service';
import {EnvironmentAuditService} from '../../../service/environment/environment.audit.service';
import {ApplicationAuditService} from '../../../service/application/application.audit.service';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';

describe('CDS: Variable List Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
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
                ApplicationAuditService
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });
    });


    it('Load Component + update value', fakeAsync(  () => {
        const http = TestBed.get(HttpTestingController);

        let mock = ['string', 'password'];

        // Create component
        let fixture = TestBed.createComponent(VariableComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/variable/type';
        })).flush(mock);

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
    }));
});

