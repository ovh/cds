/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, tick, getTestBed} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {ActionStepFormComponent} from './step.form.component';
import {SharedService} from '../../../shared.service';
import {ParameterService} from '../../../../service/parameter/parameter.service';
import {SharedModule} from '../../../shared.module';
import {Action} from '../../../../model/action.model';
import {StepEvent} from '../step.event';


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


    it('should send add step event', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(ActionStepFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();


        let step = new Action();
        step.final = true;
        fixture.componentInstance.step = step;
        fixture.componentInstance.publicActions = new Array<Action>();
        let a = new Action();
        a.name = 'Script';
        fixture.componentInstance.publicActions.push(a);

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.create, 'emit');


        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.addbtn.ui.right.floated.button')).toBeTruthy('Add button must be displayed');
        compiled.querySelector('.addbtn.ui.right.floated.button').click();

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
