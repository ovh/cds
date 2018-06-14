/* tslint:disable:no-unused-variable */

import {TestBed, fakeAsync, tick} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {RouterTestingModule} from '@angular/router/testing';

import {UserService} from '../../../service/user/user.service';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {AppModule} from '../../../app.module';
import {Router, ActivatedRoute} from '@angular/router';
import {VerifyComponent} from './verify.component';
import {AccountModule} from '../account.module';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';

describe('CDS: VerifyComponent', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                UserService,
                AuthentificationStore,
                { provide: Router, useClass: MockRouter},
                { provide: ActivatedRoute, useClass: MockActivatedRoutes}
            ],
            imports : [
                AppModule,
                RouterTestingModule.withRoutes([]),
                AccountModule,
                HttpClientTestingModule
            ]
        });
    });

    it('Verify OK', fakeAsync( () => {
        const http = TestBed.get(HttpTestingController);

        let mock = {
            'user': {
                'username': 'foo'
            },
            'password': 'bar'
        };

        // Create component
        let fixture = TestBed.createComponent(VerifyComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.ngOnInit();

        const req = http.expectOne('http://localhost:8081/user/foo/confirm/myToken');
        req.flush(mock);

        expect(fixture.componentInstance.showErrorMessage).toBeFalsy('We must not show error message is activation is ok');

        fixture.detectChanges();
        tick(250);
    }));
});

export class MockRouter {
    public navigate() {
    }
}

export class MockActivatedRoutes {
    snapshot = {
        params: {
            'username': 'foo',
            'token': 'myToken'
        }
    };
}
