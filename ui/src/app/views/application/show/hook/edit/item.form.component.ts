import {Component, Input} from '@angular/core';
import {Application} from '../../../../../model/application.model';
import {Hook} from '../../../../../model/hook.model';
import {Project} from '../../../../../model/project.model';

@Component({
    selector: 'app-application-hook-item-form',
    templateUrl: './item.form.html',
    styleUrls: ['./item.form.scss']
})
export class ApplicationHookItemFormComponent {

    @Input() project: Project;
    @Input() application: Application;
    @Input() hook: Hook;

    constructor() { }

    toggleHook(current: boolean) {
        this.hook.hasChanged = true;
        return !current;
    }
}
