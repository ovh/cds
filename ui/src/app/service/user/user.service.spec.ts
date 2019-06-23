import { APP_BASE_HREF } from '@angular/common';
import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { Injector } from '@angular/core';
import { async, getTestBed, TestBed } from '@angular/core/testing';
import { Response, ResponseOptions } from '@angular/http';
import { MockBackend } from '@angular/http/testing';
import { RouterModule } from '@angular/router';
import { AppModule } from 'app/app.module';
import { User } from 'app/model/user.model';
import { ThemeStore } from '../services.module';
import { UserService } from './user.service';

describe('CDS: User Service + Authent Store', () => {
    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                AuthentificationStore,
                UserService,
                ThemeStore
            ],
            imports: [
                AppModule,
                RouterModule,
                HttpClientTestingModule
            ]
        });

    });

    it('login', async(() => {
        const http = TestBed.get(HttpTestingController);
        let connectedChecked = false;
        let started = false;

        // user to create
        let u = new User();
        u.id = 1;
        u.username = 'foo';

        let loginResponse = {
            'user': {
                'id': 1,
                'username': 'foo'
            }
        };

        let authenStore = TestBed.get(AuthentificationStore);
        // Assertion
        authenStore.getUserlst().subscribe(user => {
            if (started) {
                connectedChecked = true;
                expect(user.id).toBe(1, 'Wrong user id');
                expect(authenStore.isConnected).toBeTruthy('User must be connected');
            }
        });

        // Begin test
        started = true;
        let userService = TestBed.get(UserService);
        userService.login(u).subscribe(() => { });
        http.expectOne('http://localhost:8081/login').flush(loginResponse);


        // Final assertion
        expect(connectedChecked).toBeTruthy('User never connected');
    }));
});
