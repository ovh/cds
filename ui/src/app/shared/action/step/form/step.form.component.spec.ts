/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, tick, getTestBed} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {ActionStepFormComponent} from './step.form.component';
import {SharedService} from '../../../shared.service';
import {ParameterService} from '../../../../service/parameter/parameter.service';
import {SharedModule} from '../../../shared.module';
import {Action} from '../../../../model/action.model';
import {StepEvent} from '../step.event';


describe('CDS: Step Form Component', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                TranslateService,
                ParameterService,
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });


    it('should send add step event', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ActionStepFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();


        let step = new Action();
        step.always_executed = true;
        fixture.componentInstance.step = step;
        fixture.componentInstance.publicActions = new Array<Action>();
        let a = new Action();
        a.name = 'Script';
        fixture.componentInstance.publicActions.push(a);

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.create, 'emit');


        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.ui.fluid.blue.button')).toBeTruthy('Add button must be displayed');
        compiled.querySelector('.ui.fluid.blue.button').click();

        fixture.detectChanges();
        tick(50);

        expect(fixture.componentInstance.create.emit).toHaveBeenCalledWith(
            new StepEvent('displayChoice', null)
        );

        expect(compiled.querySelector('.ui.green.button')).toBeTruthy('Add green button must be displayed');
        compiled.querySelector('.ui.green.button').click();

        fixture.detectChanges();
        tick(50);

        expect(fixture.componentInstance.create.emit).toHaveBeenCalledWith(
            new StepEvent('add', fixture.componentInstance.step)
        );
    }));
});
