/* tslint:disable:no-unused-variable */

import {TestBed, tick, fakeAsync} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {TranslateService, TranslateLoader, TranslateParser} from '@ngx-translate/core';
import {SharedService} from '../shared.service';
import {SharedModule} from '../shared.module';
import {UsageComponent} from './usage.component';
import {Workflow} from '../../model/workflow.model';
import {User} from '../../model/user.model';
import {Application} from '../../model/application.model';
import {Environment} from '../../model/environment.model';

describe('CDS: Usage Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                SharedService,
                TranslateService,
                TranslateLoader,
                TranslateParser,
                // Router,
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });
    });

    it('should display workflows linked', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(UsageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let fakeUser = new User();
        fakeUser.admin = true;

        fixture.componentInstance.workflows = [new Workflow(), new Workflow()];

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('app-usage-workflows')).toBeTruthy('workflows not rendered');
    }));

    it('should display applications and environments', fakeAsync( () => {
        let fixture = TestBed.createComponent(UsageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.applications = [new Application(), new Application()];
        fixture.componentInstance.environments = [new Environment(), new Environment()];
        fixture.componentInstance.workflows = [];


        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('app-usage-applications')).toBeTruthy('applications not rendered');
        expect(compiled.querySelector('app-usage-environments')).toBeTruthy('environments not rendered');
        expect(compiled.querySelector('app-usage-workflows')).toBeFalsy('workflows rendered but should not');
    }));
});
