import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { fakeAsync, inject, TestBed, tick } from '@angular/core/testing';
import { XHRBackend } from '@angular/http';
import { MockBackend } from '@angular/http/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { Group, GroupPermission } from '../../../model/group.model';
import { GroupService } from '../../../service/group/group.service';
import { SharedModule } from '../../shared.module';
import { PermissionEvent } from '../permission.event.model';
import { PermissionService } from '../permission.service';
import { PermissionFormComponent } from './permission.form.component';

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
            imports: [
                SharedModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });

    });


    it('should create new permission', fakeAsync(inject([XHRBackend], (backend: MockBackend) => {
        const http = TestBed.get(HttpTestingController);

        let groupsMock = new Array<Group>();

        let groupMock = new Group();
        groupMock.id = 1;
        groupMock.name = 'grp1';
        groupMock.members = [];

        groupsMock.push(groupMock);

        // Create component
        let fixture = TestBed.createComponent(PermissionFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/group';
        })).flush(groupsMock);

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

