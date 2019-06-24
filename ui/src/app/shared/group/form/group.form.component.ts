import {ChangeDetectionStrategy, Component, Input} from '@angular/core';
import {Group} from '../../../model/group.model';

@Component({
    selector: 'app-group-form',
    templateUrl: './group.form.html',
    styleUrls: ['./group.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GroupFormComponent {
    @Input() group: Group;
    // form ( with submit button ), noform ( no button, use group input )
    @Input() mode = 'form';

    constructor() {
        this.group = new Group();
    }

    createGroup(): void {
        // TODO later
    }
}
