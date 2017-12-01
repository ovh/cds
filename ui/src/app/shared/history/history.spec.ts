/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {SharedModule} from '../shared.module';
import {HistoryComponent} from './history.component';
import {PipelineBuild, PipelineBuildTrigger} from '../../model/pipeline.model';
import {User} from '../../model/user.model';
import {ApplicationPipelineService} from '../../service/application/pipeline/application.pipeline.service';
import {ToasterService} from 'angular2-toaster';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: History component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser,
                ApplicationPipelineService,
                ToasterService
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

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

        expect(fixture.componentInstance.getTriggerSource(pb)).toBe('CDS scheduler');
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

        expect(fixture.componentInstance.getTriggerSource(pb)).toBe('Foo');
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

        expect(fixture.componentInstance.getTriggerSource(pb)).toBe('Bar');
    }));
});

