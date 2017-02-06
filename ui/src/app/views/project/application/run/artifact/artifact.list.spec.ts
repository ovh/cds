/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed, tick} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {SharedModule} from '../../../../../shared/shared.module';
import {ArtifactListComponent} from './artifact.list.component';
import {Observable} from 'rxjs';
import {CDSWorker} from '../../../../../shared/worker/worker';
import {ApplicationRunModule} from '../application.run.module';

describe('CDS: Artifact List', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser,
            ],
            imports: [
                ApplicationRunModule,
                RouterTestingModule.withRoutes([]),
                SharedModule
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });

    it('should load component', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(ArtifactListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.buildWorker = new MockWorker();
        fixture.componentInstance.ngOnInit();

        fixture.detectChanges();
        tick(50);

        expect(fixture.componentInstance.artifacts.length).toBe(2);

    }));
});

class MockWorker extends CDSWorker {
    constructor() {
        super('fake', 'fake');
    }

    response(): any {
        let response = { data : '{ "artifacts": [ { "name": "art1" }, { "name": "art2" }] }' };
        return Observable.of(response);
    }
}
