/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, fakeAsync, tick} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {SharedModule} from '../../shared.module';
import {GroupFormComponent} from './group.form.component';

describe('CDS: Group form component', () => {

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
                TranslateParser,
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


    it('should load component and disable button', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(GroupFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(250);

        expect(fixture.debugElement.nativeElement.querySelector('.ui.green.button.disabled')).toBeTruthy();
    }));

    it('should load component and enable button', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(GroupFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.group.name = 'My New Group';

        fixture.detectChanges();
        tick(250);
        expect(fixture.debugElement.nativeElement.querySelector('.ui.green.button.disabled')).toBeFalsy();
    }));


});

