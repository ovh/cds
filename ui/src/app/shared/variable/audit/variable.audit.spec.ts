/* eslint-disable @typescript-eslint/no-unused-vars */
import {TestBed, fakeAsync, tick} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {RouterTestingModule} from '@angular/router/testing';
import {APP_BASE_HREF} from '@angular/common';
import {SharedService} from '../../shared.service';
import {VariableAudit} from '../../../model/variable.model';
import {SharedModule} from '../../shared.module';
import {VariableAuditComponent} from './audit.component';

describe('CDS: Variable Audit Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                TranslateService,
                TranslateLoader,
                TranslateParser,
                { provide: APP_BASE_HREF, useValue : '/' }
            ],
            imports : [
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        }).compileComponents();

    });

    it('Load Component', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(VariableAuditComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.audits = new Array<VariableAudit>();
        let vac = new VariableAudit();
        vac.type = 'add';

        fixture.detectChanges();
        tick(50);
    }));
});

