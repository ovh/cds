import { ChangeDetectionStrategy, Component } from '@angular/core';

@Component({
    standalone: false,
    selector: 'app-workflow-template-help',
    templateUrl: './workflow-template.help.html',
    styleUrls: ['./workflow-template.help.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowTemplateHelpComponent { }
