/* eslint-disable @typescript-eslint/no-unused-vars */

import { APP_BASE_HREF } from '@angular/common';
import { fakeAsync, flush, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Action } from '../../../../model/action.model';
import { ParameterService } from '../../../../service/parameter/parameter.service';
import { SharedModule } from '../../../shared.module';
import { SharedService } from '../../../shared.service';
import { StepEvent } from '../step.event';
import { ActionStepFormComponent } from './step.form.component';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

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
                BrowserModule,
                BrowserAnimationsModule,
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
        fixture.componentInstance.actions = [];

        let scriptAction = new Action();
        scriptAction.name = 'Script';
        fixture.componentInstance.actions.push(scriptAction);
        fixture.componentInstance.selected = scriptAction;

        expect(component).toBeTruthy();

        fixture.detectChanges();

        spyOn(fixture.componentInstance.onEvent, 'emit');


        let compiled = fixture.debugElement.nativeElement;
        let button = compiled.querySelector('button[name="addStepMenuBtn"]');
        console.log(button);
        expect(button).toBeTruthy('Add step menu btn must be displayed');
        button.click();

        expect(fixture.componentInstance.onEvent.emit).toHaveBeenCalledWith(
            new StepEvent('expend', null)
        );

        fixture.changeDetectorRef.markForCheck();
        tick(200);

        console.log(compiled);
        expect(compiled.querySelector('button[name="addStepBtn"]')).toBeTruthy('Add step button must be displayed');
        compiled.querySelector('button[name="addStepBtn"]').click();

        expect(fixture.componentInstance.onEvent.emit).toHaveBeenCalledWith(
            new StepEvent('add', fixture.componentInstance.selected)
        );

        flush();
    }));
});
