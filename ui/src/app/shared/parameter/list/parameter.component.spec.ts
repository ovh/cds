/* tslint:disable:no-unused-variable */

import {TestBed, tick, fakeAsync, inject} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {SharedModule} from '../../shared.module';
import {ParameterService} from '../../../service/parameter/parameter.service';
import {ParameterListComponent} from './parameter.component';
import {Parameter} from '../../../model/parameter.model';
import {ParameterEvent} from '../parameter.event.model';
import {RepoManagerService} from '../../../service/repomanager/project.repomanager.service';

describe('CDS: Parameter List Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                ParameterService,
                TranslateService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser,
                RepoManagerService,
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });
    });


    it('should load component + update description', fakeAsync(
        inject([XHRBackend], (backend: MockBackend) => {
        // Mock Http request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '["string", "password"]'})));
        });

        // Create component
        let fixture = TestBed.createComponent(ParameterListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        expect(backend.connectionsArray[0].request.url).toBe('/parameter/type', 'Component must load parameter type');

        let params: Parameter[] = [];
        let p: Parameter = new Parameter();
        p.name = 'foo';
        p.type = 'string';
        p.description = 'foo is my param';

        params.push(p);
        fixture.componentInstance.parameters = params;

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
            new ParameterEvent('delete', fixture.componentInstance.parameters[0])
        );
    })));
});

