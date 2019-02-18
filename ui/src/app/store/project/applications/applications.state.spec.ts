import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { async, TestBed } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { FetchApplication } from './applications.action';
import { ApplicationsState } from './applications.state';

describe('Applications', () => {
    let store: Store;

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [
                NgxsModule.forRoot([ApplicationsState]),
                HttpClientTestingModule
            ],
        }).compileComponents();

        store = TestBed.get(Store);
        // store.reset(getInitialApplicationsState());
    }));

    it('it fetch application', async(() => {
        const http = TestBed.get(HttpTestingController);
        store.dispatch(new FetchApplication({
            projectKey: 'test1',
            applicationName: 'app1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });
    }));
});
