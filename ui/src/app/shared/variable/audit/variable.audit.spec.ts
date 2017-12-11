/* tslint:disable:no-unused-variable */
import {TestBed, getTestBed, fakeAsync, tick} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from '@ngx-translate/core';
import {SharedService} from '../../shared.service';
import {RouterTestingModule} from '@angular/router/testing';
import {VariableAudit} from '../../../model/variable.model';
import {Injector} from '@angular/core';
import {SharedModule} from '../../shared.module';
import {VariableAuditComponent} from './audit.component';

describe('CDS: Variable Audit Component', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                TranslateService,
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

