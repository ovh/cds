/* eslint-disable @typescript-eslint/no-unused-vars */

import { APP_BASE_HREF } from '@angular/common';
import { fakeAsync, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Action } from '../../../../model/action.model';
import { ParameterService } from '../../../../service/parameter/parameter.service';
import { SharedModule } from '../../../shared.module';
import { SharedService } from '../../../shared.service';
import { StepEvent } from '../step.event';
import { ActionStepFormComponent } from './step.form.component';

describe('CDS: Step Form Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                TranslateService,
                ParameterService,
                TranslateLoader,
                TranslateParser,
                { provide: APP_BASE_HREF, useValue: '/' }
            ],
            imports: [
                RouterTestingModule.withRoutes([]),
                TranslateModule.forRoot(),
                SharedModule
            ]
        }).compileComponents();
    });

    it('should send add step event', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ActionStepFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();


        let step = new Action();
        step.always_executed = true;
        fixture.componentInstance.selected = step;
        fixture.componentInstance.actions = new Array<Action>();
        let a = new Action();
        a.name = 'Script';
        fixture.componentInstance.actions.push(a);

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.onEvent, 'emit');


        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('.ui.fluid.blue.button')).toBeTruthy('Add button must be displayed');
        compiled.querySelector('.ui.fluid.blue.button').click();

        fixture.detectChanges();
        tick(50);

        expect(fixture.componentInstance.onEvent.emit).toHaveBeenCalledWith(
            new StepEvent('expend', null)
        );

        expect(compiled.querySelector('.ui.green.button')).toBeTruthy('Add green button must be displayed');
        compiled.querySelector('.ui.green.button').click();

        fixture.detectChanges();
        tick(50);

        expect(fixture.componentInstance.onEvent.emit).toHaveBeenCalledWith(
            new StepEvent('add', fixture.componentInstance.selected)
        );
    }));
});
