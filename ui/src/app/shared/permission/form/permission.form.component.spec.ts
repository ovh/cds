/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync, inject} from '@angular/core/testing';
import {TranslateService, TranslateLoader, TranslateParser} from 'ng2-translate';
import {RouterTestingModule} from '@angular/router/testing';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {GroupService} from '../../../service/group/group.service';
import {PermissionFormComponent} from './permission.form.component';
import {GroupPermission} from '../../../model/group.model';
import {PermissionService} from '../permission.service';
import {PermissionEvent} from '../permission.event.model';
import {SharedModule} from '../../shared.module';

describe('CDS: Permission From Component', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                GroupService,
                PermissionService,
                TranslateService,
                { provide: XHRBackend, useClass: MockBackend },
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([])
            ]
        });

    });


    it('should create new permission', fakeAsync( inject([XHRBackend], (backend: MockBackend) => {
        // Mock Http login request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({ body : '[ { "id": 1, "name": "grp1", "admins": [], "users": [] },' +
            ' { "id": 2, "name": "grp2", "users": [], "admins": []  }]'})));
        });


        // Create component
        let fixture = TestBed.createComponent(PermissionFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(50);

        expect(fixture.debugElement.nativeElement.querySelector('.ui.button.disabled')).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        // Permission to add
        let gp = new GroupPermission();
        gp.group.name = 'grp1';
        gp.permission = 7;

        fixture.detectChanges();
        tick(50);

        // Emulate typing
        fixture.componentInstance.newGroupPermission = gp;

        // Click on create button
        spyOn(fixture.componentInstance.createGroupPermissionEvent, 'emit');
        compiled.querySelector('.ui.green.button').click();

        // Check if creation evant has been emitted
        expect(fixture.componentInstance.createGroupPermissionEvent.emit).toHaveBeenCalledWith(new PermissionEvent('add', gp));
    })));
});

