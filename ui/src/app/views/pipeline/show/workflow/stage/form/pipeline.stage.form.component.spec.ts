/* tslint:disable:no-unused-variable */
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { fakeAsync, TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { NgxsStoreModule } from 'app/store/store.module';
import 'rxjs/add/observable/of';
import { Observable } from 'rxjs/Observable';
import { Prerequisite } from '../../../../../../model/prerequisite.model';
import { Stage } from '../../../../../../model/stage.model';
import { PrerequisiteEvent } from '../../../../../../shared/prerequisites/prerequisite.event.model';
import { SharedModule } from '../../../../../../shared/shared.module';
import { PipelineModule } from '../../../../pipeline.module';
import { PipelineStageFormComponent } from './pipeline.stage.form.component';


describe('CDS: Stage From component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: ActivatedRoute, useClass: MockActivatedRoutes },
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports: [
                PipelineModule,
                NgxsStoreModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });
    });

    it('should add and delete prerequisite', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(PipelineStageFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Init stage
        let s = new Stage();
        fixture.componentInstance.stage = s;

        let eventAdd = new PrerequisiteEvent('add', new Prerequisite());
        eventAdd.prerequisite.parameter = 'git.branch';
        eventAdd.prerequisite.expected_value = 'master';

        fixture.componentInstance.prerequisiteEvent(eventAdd);
        // add twice
        fixture.componentInstance.prerequisiteEvent(eventAdd);

        expect(fixture.componentInstance.stage.prerequisites.length).toBe(1, 'Must have 1 prerequisite');
        expect(fixture.componentInstance.stage.prerequisites[0].parameter).toBe('git.branch');
        expect(fixture.componentInstance.stage.prerequisites[0].expected_value).toBe('master');


        let eventDelete = new PrerequisiteEvent('delete', eventAdd.prerequisite);
        fixture.componentInstance.prerequisiteEvent(eventDelete);
        expect(fixture.componentInstance.stage.prerequisites.length).toBe(0, 'Must have 0 prerequisite');
    }));
});

class MockToast {
    success(title: string, msg: string) {

    }
}

class MockActivatedRoutes extends ActivatedRoute {
    constructor() {
        super();
        this.params = Observable.of({ key: 'key1', appName: 'app1' });
        this.queryParams = Observable.of({ key: 'key1', appName: 'app1' });
    }
}
