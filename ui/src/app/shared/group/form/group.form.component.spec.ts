/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, tick} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {SharedModule} from '../../shared.module';
import {GroupFormComponent} from './group.form.component';

describe('CDS: Group form component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser,
            ],
            imports : [
                SharedModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([])
            ]
        });

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

