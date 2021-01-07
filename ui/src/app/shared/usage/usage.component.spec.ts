/* eslint-disable @typescript-eslint/no-unused-vars */

import {TestBed} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {Component, NO_ERRORS_SCHEMA} from '@angular/core';
import {SharedService} from '../shared.service';
import {SharedModule} from '../shared.module';
import {Workflow} from '../../model/workflow.model';
import {Application} from '../../model/application.model';
import {Environment} from '../../model/environment.model';
import {Project} from '../../model/project.model';
import {UsageComponent} from './usage.component';

describe('CDS: Usage Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                DummyComponent
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
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([
                    { path: 'project/:key/workflow/:name', component: DummyComponent},
                    { path: 'project/:key/application/:name', component: DummyComponent},
                    { path: 'project/:key/pipeline/:name', component: DummyComponent},
                    { path: 'project/:key/environment/:name', component: DummyComponent}
                ])
            ],
            schemas: [ NO_ERRORS_SCHEMA ]
        }).compileComponents();
    });

    it('should display workflows linked', () => {
        // Create component
        let fixture = TestBed.createComponent(UsageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let p = new Project();
        p.key = 'key';
        fixture.componentInstance.project = p;
        fixture.componentInstance.workflows = [new Workflow(), new Workflow()];

        fixture.detectChanges(true);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('app-usage-workflows')).toBeTruthy('workflows not rendered');

    });

    it('should display applications and environments', () => {
        let fixture = TestBed.createComponent(UsageComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let p = new Project();
        p.key = 'key';
        fixture.componentInstance.project = p;
        fixture.componentInstance.applications = [new Application(), new Application()];
        fixture.componentInstance.environments = [new Environment(), new Environment()];
        fixture.componentInstance.workflows = [];

        fixture.detectChanges(true);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelector('app-usage-applications')).toBeTruthy('applications not rendered');
        expect(compiled.querySelector('app-usage-environments')).toBeTruthy('environments not rendered');
        expect(compiled.querySelector('app-usage-workflows')).toBeFalsy('workflows rendered but should not');

    });
});


@Component({
    template: '',
})
export class DummyComponent {
}
