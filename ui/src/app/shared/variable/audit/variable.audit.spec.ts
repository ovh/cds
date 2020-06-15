/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, tick} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {SharedService} from '../../shared.service';
import {RouterTestingModule} from '@angular/router/testing';
import {VariableAudit} from '../../../model/variable.model';
import {SharedModule} from '../../shared.module';
import {VariableAuditComponent} from './audit.component';
import {APP_BASE_HREF} from '@angular/common';

describe('CDS: Variable Audit Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
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
        });

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

