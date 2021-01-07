/* eslint-disable @typescript-eslint/no-unused-vars */

import { fakeAsync, TestBed, tick } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { TranslateLoader, TranslateModule, TranslateParser, TranslateService } from '@ngx-translate/core';
import { GroupPermission } from '../../../model/group.model';
import { SharedModule } from '../../shared.module';
import { SharedService } from '../../shared.service';
import { PermissionEvent } from '../permission.event.model';
import { PermissionService } from '../permission.service';
import { PermissionListComponent } from './permission.list.component';

describe('CDS: Permission List Component', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                PermissionService,
                SharedService,
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                SharedModule,
                RouterTestingModule.withRoutes([]),
                TranslateModule.forRoot()
            ]
        }).compileComponents();
    });

    it('should delete a permission', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(PermissionListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.edit = true;

        // Init array of permissions
        let groupsPermission: GroupPermission[] = [];
        let gp: GroupPermission = new GroupPermission();
        gp.group.name = 'group1';
        gp.permission = 7;
        groupsPermission.push(gp);

        fixture.componentInstance.permissions = groupsPermission;

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;

        // Click on delete button

        expect(compiled.querySelector('.ui.red.button')).toBeTruthy('Delete button must be displayed');
        compiled.querySelector('.ui.red.button').click();

        fixture.detectChanges();
        tick(50);

        spyOn(fixture.componentInstance.event, 'emit');

        // Click on delete confirmation

        expect(compiled.querySelector('.ui.buttons')).toBeTruthy('Confirmation buttons must be displayed');
        compiled.querySelector('.ui.red.button.active').click();

        // Check if delete event has been emitted
        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(new PermissionEvent('delete', gp));

    }));

    it('should update a permission', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(PermissionListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.componentInstance.edit = true;

        // Init array of permissions
        let groupsPermission: GroupPermission[] = [];
        let gp: GroupPermission = new GroupPermission();
        gp.group.name = 'group1';
        gp.permission = 7;
        gp.hasChanged = true;
        groupsPermission.push(gp);

        fixture.componentInstance.permissions = groupsPermission;

        fixture.detectChanges();
        tick(50);

        let compiled = fixture.debugElement.nativeElement;

        // Click on update button
        spyOn(fixture.componentInstance.event, 'emit');
        expect(compiled.querySelector('.ui.red.button')).toBeFalsy('No delete button, update case');
        compiled.querySelector('button[name="btnupdateperm"]').click();

        // Check if delete event has been emitted
        expect(fixture.componentInstance.event.emit).toHaveBeenCalledWith(new PermissionEvent('update', gp));
    }));

    it('should get permission name by value', fakeAsync( () => {
        // Create component
        let fixture = TestBed.createComponent(PermissionListComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        expect(fixture.componentInstance.getPermissionName(7)).toBe('permission_read_write_execute');
    }));
});

