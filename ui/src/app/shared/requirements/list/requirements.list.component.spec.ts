/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {SharedModule} from '../../shared.module';
import {RequirementService} from '../../../service/worker/requirement/requirement.service';
import {RequirementsListComponent} from './requirements.list.component';
import {Requirement} from '../../../model/requirement.model';
import {RequirementEvent} from '../requirement.event.model';
import {RequirementStore} from '../../../service/worker/requirement/requirement.store';
import {WorkerModelService} from '../../../service/worker/worker.model.service';

describe('CDS: Requirement List Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateParser,
                RequirementService,
                TranslateService,
                RequirementStore,
                WorkerModelService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
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


    it('should load component + delete requirement', fakeAsync( () => {
        // Mock Http request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '["binary", "network"]'})));
        });

        // Create component
        let fixture = TestBed.createComponent(RequirementsListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();


        expect(backend.connectionsArray[0].request.url).toBe('/worker/model/capability/type', 'Component must load requirement type');
        expect(JSON.stringify(fixture.componentInstance.availableRequirements)).toBe(JSON.stringify(['binary', 'network']));

        let reqs: Requirement[] = [];
        let r: Requirement = new Requirement('binary');
        r.name = 'foo';
        r.value = 'bar';

        reqs.push(r);
        fixture.componentInstance.requirements = reqs;

        // Readonly mode: no delete button displayed
        expect(fixture.debugElement.nativeElement.querySelector('.ui.red.button')).toBeFalsy();

        fixture.componentInstance.edit = true;

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
            new RequirementEvent('delete', fixture.componentInstance.requirements[0])
        );
    }));
});

