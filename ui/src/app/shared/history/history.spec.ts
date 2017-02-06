/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend} from '@angular/http';
import {Injector} from '@angular/core';
import {SharedModule} from '../shared.module';
import {HistoryComponent} from './history.component';
import {PipelineBuild, PipelineBuildTrigger} from '../../model/pipeline.model';
import {User} from '../../model/user.model';

describe('CDS: History component', () => {

    let injector: Injector;
    let backend: MockBackend;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser,
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

        injector = getTestBed();
        backend = injector.get(XHRBackend);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
    });


    it('should return that pipeline was triggered by CDS scheduler', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(HistoryComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let pb = new PipelineBuild();
        pb.trigger = new PipelineBuildTrigger();
        pb.trigger.scheduled_trigger = true;
        pb.trigger.triggered_by = new User();
        pb.trigger.triggered_by.username = 'Foo';
        pb.trigger.vcs_author = 'Bar';

        expect(fixture.componentInstance.getTriggerSource(pb)).toBe('CDS scheduler')
    }));

    it('should return that pipeline was triggered by a CDS User', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(HistoryComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let pb = new PipelineBuild();
        pb.trigger = new PipelineBuildTrigger();
        pb.trigger.scheduled_trigger = false;
        pb.trigger.triggered_by = new User();
        pb.trigger.triggered_by.username = 'Foo';
        pb.trigger.vcs_author = 'Bar';

        expect(fixture.componentInstance.getTriggerSource(pb)).toBe('Foo')
    }));

    it('should return that pipeline was triggered by Git commit author', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(HistoryComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let pb = new PipelineBuild();
        pb.trigger = new PipelineBuildTrigger();
        pb.trigger.scheduled_trigger = false;
        pb.trigger.triggered_by = new User();
        pb.trigger.triggered_by.username = '';
        pb.trigger.vcs_author = 'Bar';

        expect(fixture.componentInstance.getTriggerSource(pb)).toBe('Bar')
    }));
});

