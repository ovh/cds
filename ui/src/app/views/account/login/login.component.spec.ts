/* tslint:disable:no-unused-variable */

import {TestBed,  tick, fakeAsync} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {RouterTestingModule} from '@angular/router/testing';
import {LoginComponent} from './login.component';

import {UserService} from '../../../service/user/user.service';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {AppModule} from '../../../app.module';
import {Router, ActivatedRoute} from '@angular/router';
import {AccountModule} from '../account.module';

import {Observable} from 'rxjs/Observable';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';
import 'rxjs/add/observable/of';

describe('CDS: LoginComponent', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                UserService,
                AuthentificationStore,
                { provide: Router, useClass: MockRouter},
                { provide: ActivatedRoute, useValue: { queryParams: Observable.of({redirection: null})} },
            ],
            imports : [
                AppModule,
                RouterTestingModule.withRoutes([]),
                AccountModule,
                HttpClientTestingModule
            ]
        });
    });


    it('Click on Login button', fakeAsync(() => {
        const http = TestBed.get(HttpTestingController);

        let mock = {
            'user' : {
                'username': 'foo'
            }
        };

        // Create loginComponent
        let fixture = TestBed.createComponent(LoginComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        // Start detecting change in model
        fixture.detectChanges();
        tick(50);

        // Simulate user typing
        let inputUsername = compiled.querySelector('input[name="username"]');
        inputUsername.value = 'foo';
        inputUsername.dispatchEvent(new Event('input'));

        let inputPassword = compiled.querySelector('input[name="password"]');
        inputPassword.value = 'bar';
        inputPassword.dispatchEvent(new Event('input'));

        // Simulate user click
        compiled.querySelector('#loginButton').click();
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/login' && req.body.username === 'foo' && req.body.password === 'bar';
        })).flush(mock);

        http.verify();
    }));
});

export class MockRouter {
    public navigate() {
    }
}
