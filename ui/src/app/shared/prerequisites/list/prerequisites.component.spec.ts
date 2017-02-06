/* tslint:disable:no-unused-variable */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {SharedModule} from '../../shared.module';
import {PrerequisiteComponent} from './prerequisites.component';
import {Prerequisite} from '../../../model/prerequisite.model';
import {PrerequisiteEvent} from '../prerequisite.event.model';

describe('CDS: Prerequisite List Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });


    it('should load component + update value', fakeAsync( () => {

        // Create component
        let fixture = TestBed.createComponent(PrerequisiteComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let pres: Prerequisite[] = [];
        let p: Prerequisite = new Prerequisite();
        p.parameter = 'foo';
        p.expected_value = 'bar';

        pres.push(p);
        fixture.componentInstance.prerequisites = pres;

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
            new PrerequisiteEvent('delete', pres[0])
        );
    }));
});

