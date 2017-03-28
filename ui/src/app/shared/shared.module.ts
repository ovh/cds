import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {VariableComponent} from './variable/list/variable.component';
import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import {TranslateModule} from 'ng2-translate';
import {PrettyJsonModule} from 'angular2-prettyjson';
import {NgSemanticModule} from 'ng-semantic/ng-semantic';
import {NgForNumber} from './pipes/ngfor.number.pipe';
import {VariableValueComponent} from './variable/value/variable.value.component';
import {VariableFormComponent} from './variable/form/variable.form';
import {SharedService} from './shared.service';
import {PermissionService} from './permission/permission.service';
import {PermissionListComponent} from './permission/list/permission.list.component';
import {PermissionFormComponent} from './permission/form/permission.form.component';
import {DeleteButtonComponent} from './button/delete/delete.button';
import {ToastService} from './toast/ToastService';
import {BreadcrumbComponent} from './breadcrumb/breadcrumb.component';
import {ActionComponent} from './action/action.component';
import {PrerequisiteComponent} from './prerequisites/list/prerequisites.component';
import {PrerequisitesFormComponent} from './prerequisites/form/prerequisites.form.component';
import {RequirementsListComponent} from './requirements/list/requirements.list.component';
import {RequirementsFormComponent} from './requirements/form/requirements.form.component';
import {ParameterListComponent} from './parameter/list/parameter.component';
import {ParameterFormComponent} from './parameter/form/parameter.form';
import {ParameterValueComponent} from './parameter/value/parameter.value.component';
import {DragulaModule} from 'ng2-dragula/ng2-dragula';
import {WarningModalComponent} from './modal/warning/warning.component';
import {CommonModule} from '@angular/common';
import {CutPipe} from './pipes/cut.pipe';
import {MomentModule} from 'angular2-moment';
import {CodemirrorModule} from 'ng2-codemirror';
import {GroupFormComponent} from './group/form/group.form.component';
import {MarkdownModule} from 'angular2-markdown';
import {HistoryComponent} from './history/history.component';
import {StatusIconComponent} from './status/status.component';
import {KeysPipe} from './pipes/keys.pipe';
import {DurationService} from './duration/duration.service';
import {ParameterDescriptionComponent} from './parameter/description-popup/description.popup.component';
import {ActionStepComponent} from './action/step/step.component';
import {ActionStepFormComponent} from './action/step/form/step.form.component';
import {TruncatePipe} from './pipes/truncate.pipe';
import {VariableAuditComponent} from './variable/audit/audit.component';
import {VariableDiffComponent} from './variable/diff/variable.diff.component';
import {ZoneContentComponent} from './zone/zone-content/content.component';
import {ZoneComponent} from './zone/zone.component';

@NgModule({
    imports: [ CommonModule, NgSemanticModule, FormsModule, TranslateModule, DragulaModule, MomentModule,
        PrettyJsonModule, CodemirrorModule, ReactiveFormsModule, MarkdownModule ],
    declarations: [
        ActionComponent,
        ActionStepComponent,
        ActionStepFormComponent,
        BreadcrumbComponent,
        CutPipe,
        DeleteButtonComponent,
        GroupFormComponent,
        HistoryComponent,
        KeysPipe,
        NgForNumber,
        ParameterDescriptionComponent,
        ParameterListComponent,
        ParameterFormComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        PrerequisiteComponent,
        PrerequisitesFormComponent,
        RequirementsListComponent,
        RequirementsFormComponent,
        StatusIconComponent,
        TruncatePipe,
        VariableComponent,
        VariableAuditComponent,
        VariableDiffComponent,
        VariableFormComponent,
        VariableValueComponent,
        WarningModalComponent,
        ZoneComponent,
        ZoneContentComponent
    ],
    providers: [
        DurationService,
        PermissionService,
        SharedService,
        ToastService
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ],
    exports: [
        ActionComponent,
        BreadcrumbComponent,
        CodemirrorModule,
        CommonModule,
        CutPipe,
        DeleteButtonComponent,
        FormsModule,
        GroupFormComponent,
        HistoryComponent,
        KeysPipe,
        MarkdownModule,
        MomentModule,
        NgForNumber,
        NgSemanticModule,
        ParameterDescriptionComponent,
        ParameterListComponent,
        ParameterFormComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        PrettyJsonModule,
        PrerequisiteComponent,
        PrerequisitesFormComponent,
        ReactiveFormsModule,
        StatusIconComponent,
        TranslateModule,
        TruncatePipe,
        VariableComponent,
        VariableFormComponent,
        VariableValueComponent,
        WarningModalComponent,
        ZoneComponent,
        ZoneContentComponent
    ]
})
export class SharedModule {
}
