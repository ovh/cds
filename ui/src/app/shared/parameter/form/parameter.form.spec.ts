/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {SharedModule} from '../../shared.module';
import {ParameterService} from '../../../service/parameter/parameter.service';
import {ParameterFormComponent} from './parameter.form';
import {Parameter} from '../../../model/parameter.model';
import {ParameterEvent} from '../parameter.event.model';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: parameter From Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ParameterService,
                TranslateService,
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser,
                RepoManagerService,
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


    it('should create a new parameter', fakeAsync( () => {
        let call = 0;
        // Mock Http request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '["string", "password"]'})));
        });


        // Create component
        let fixture = TestBed.createComponent(ParameterFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        let parameter = new Parameter();
        parameter.name = 'foo';
        parameter.type = 'string';
        parameter.description = 'bar';

        fixture.detectChanges();
        tick(50);

        // simulate typing new parameter
        let inputName = compiled.querySelector('input[name="name"]');
        inputName.value = parameter.name;
        inputName.dispatchEvent(new Event('input'));

        fixture.componentInstance.newParameter.type = 'string';

        let textAreaDescription = compiled.querySelector('textarea[name="description"]');
        textAreaDescription.value = parameter.description;
        textAreaDescription.dispatchEvent(new Event('input'));
        textAreaDescription.dispatchEvent(new Event('change'));

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.createParameterEvent, 'emit');
        compiled.querySelector('.ui.blue.button').click();

        parameter.value = '';
        expect(fixture.componentInstance.createParameterEvent.emit).toHaveBeenCalledWith(new ParameterEvent('add', parameter));
    }));
});
