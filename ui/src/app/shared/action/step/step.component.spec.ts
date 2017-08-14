/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, tick, getTestBed} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {SharedService} from '../../shared.service';
import {ParameterService} from '../../../service/parameter/parameter.service';
import {SharedModule} from '../../shared.module';
import {ActionStepComponent} from './step.component';
import {Action} from '../../../model/action.model';
import {StepEvent} from './step.event';


describe('CDS: Action Component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                TranslateService,
                ParameterService,
                MockBackend,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });

        injector = getTestBed();
        backend = injector.get(MockBackend);
    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });



    it('should send remove step event', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ActionStepComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let action: Action = new Action();
        action.name = 'FooAction';
        fixture.componentInstance.action = action;
        fixture.componentInstance.edit = true;

        let step = new Action();
        step.always_executed = true;
        fixture.componentInstance.step = step;

        fixture.detectChanges();
        tick(50);


        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.ui.red.button')).toBeTruthy('Delete button must be displayed');
        compiled.querySelector('.ui.red.button').click();

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.removeEvent, 'emit');

        expect(compiled.querySelector('.ui.buttons')).toBeTruthy('Confirmation buttons must be displayed');
        compiled.querySelector('.ui.red.button.active').click();

        expect(fixture.componentInstance.removeEvent.emit).toHaveBeenCalledWith(
            new StepEvent('delete', fixture.componentInstance.step)
        );
    }));
});
