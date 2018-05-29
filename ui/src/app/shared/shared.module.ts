import {NgModule, CUSTOM_ELEMENTS_SCHEMA} from '@angular/core';
import {VariableComponent} from './variable/list/variable.component';
import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import {TranslateModule} from '@ngx-translate/core';
import {NgSemanticModule} from 'ng-semantic/ng-semantic';
import {NgForNumber} from './pipes/ngfor.number.pipe';
import {TokenListComponent} from './token/list/token.list.component';
import {VariableValueComponent} from './variable/value/variable.value.component';
import {VariableFormComponent} from './variable/form/variable.form';
import {SharedService} from './shared.service';
import {BroadcastLevelService} from './broadcast/broadcast.level.service';
import {PermissionService} from './permission/permission.service';
import {PermissionListComponent} from './permission/list/permission.list.component';
import {PermissionFormComponent} from './permission/form/permission.form.component';
import {DeleteButtonComponent} from './button/delete/delete.button';
import {UploadButtonComponent} from './button/upload/upload.button.component';
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
import {DeleteModalComponent} from './modal/delete/delete.component';
import {CommonModule} from '@angular/common';
import {CutPipe} from './pipes/cut.pipe';
import {MomentModule} from 'angular2-moment';
import {CodemirrorModule} from 'ng2-codemirror-typescript/Codemirror';
import {GroupFormComponent} from './group/form/group.form.component';
import {MarkdownModule} from 'ngx-md';
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
import {PipelineLaunchModalComponent} from './pipeline/launch/pipeline.launch.modal.component';
import {CommitListComponent} from './commit/commit.list.component';
import {NguiAutoCompleteModule} from '@ngui/auto-complete';
import {WorkflowNodeComponent} from './workflow/node/workflow.node.component';
import {WorkflowNodeAddWizardComponent} from './workflow/node/wizard/node.wizard.component';
import {WorkflowTriggerComponent} from './workflow/trigger/workflow.trigger.component';
import {WorkflowNodeFormComponent} from './workflow/node/form/workflow.node.form.component';
import {WorkflowDeleteNodeComponent} from './workflow/node/delete/workflow.node.delete.component';
import {WorkflowNodeContextComponent} from './workflow/node/context/workflow.node.context.component';
import {WorkflowJoinComponent} from './workflow/join/workflow.join.component';
import {WorkflowDeleteJoinComponent} from './workflow/join/delete/workflow.join.delete.component';
import {WorkflowTriggerJoinComponent} from './workflow/join/trigger/trigger.join.component';
import {WorkflowJoinTriggerSrcComponent} from './workflow/join/trigger/src/trigger.src.component';
import {RouterModule} from '@angular/router';
import {ForMapPipe} from './pipes/map.pipe';
import {PermissionEnvironmentFormComponent} from './permission/environment/form/permission.env.form.component';
import {NgxAutoScroll} from 'ngx-auto-scroll/src/ngx-auto-scroll.directive';
import {SuiModule} from 'ng2-semantic-ui';
import {WorkflowNodeRunParamComponent} from './workflow/node/run/node.run.param.component';
import {WorkflowNodeHookFormComponent} from './workflow/node/hook/form/hook.form.component';
import {WorkflowNodeHookComponent} from './workflow/node/hook/hook.component';
import {WorkflowNodeHookDetailsComponent} from './workflow/node/hook/details/hook.details.component';
import {UsageWorkflowsComponent} from './usage/workflows/usage.workflows.component';
import {UsageApplicationsComponent} from './usage/applications/usage.applications.component';
import {UsagePipelinesComponent} from './usage/pipelines/usage.pipelines.component';
import {UsageEnvironmentsComponent} from './usage/environments/usage.environments.component';
import {UsageComponent} from './usage/usage.component';
import {WorkflowNodeConditionFormComponent} from './workflow/node/conditions/condition-form/condition.component';
import {WorkflowNodeConditionListComponent} from './workflow/node/conditions/condition-list/condition.list.component';
import {WorkflowNodeConditionsComponent} from './workflow/node/conditions/node.conditions.component';
import {DiffComponent} from './diff/diff.component';
import {SpanColoredComponent} from './diff/span-colored/span-colored.component';
import {KeysFormComponent} from './keys/form/keys.form.component';
import {KeysListComponent} from './keys/list/keys.list.component';
import {VCSStrategyComponent} from './vcs/vcs.strategy.component';
import {RepoManagerFormComponent} from './repomanager/from/repomanager.form.component';
import {ClipboardModule} from 'ngx-clipboard';
import {FavoriteCardsComponent} from './favorite-cards/favorite-cards.component';
import {WarningTabComponent} from './warning/tab/warning.tab.component';

@NgModule({
    imports: [ CommonModule, ClipboardModule, NgSemanticModule, FormsModule, TranslateModule, DragulaModule, MomentModule,
        CodemirrorModule, ReactiveFormsModule, MarkdownModule, NguiAutoCompleteModule, RouterModule, SuiModule ],
    declarations: [
        ActionComponent,
        ActionStepComponent,
        ActionStepFormComponent,
        BreadcrumbComponent,
        CommitListComponent,
        CutPipe,
        DeleteButtonComponent,
        UploadButtonComponent,
        ForMapPipe,
        GroupFormComponent,
        HistoryComponent,
        KeysPipe,
        KeysFormComponent,
        KeysListComponent,
        NgForNumber,
        TokenListComponent,
        NgxAutoScroll,
        ParameterDescriptionComponent,
        ParameterListComponent,
        ParameterFormComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        PermissionEnvironmentFormComponent,
        PrerequisiteComponent,
        PipelineLaunchModalComponent,
        PrerequisitesFormComponent,
        RequirementsListComponent,
        RequirementsFormComponent,
        RepoManagerFormComponent,
        StatusIconComponent,
        TruncatePipe,
        VariableComponent,
        VariableAuditComponent,
        VariableDiffComponent,
        VariableFormComponent,
        VariableValueComponent,
        WarningModalComponent,
        DeleteModalComponent,
        WarningTabComponent,
        WorkflowNodeComponent,
        WorkflowNodeAddWizardComponent,
        WorkflowNodeRunParamComponent,
        WorkflowDeleteJoinComponent,
        WorkflowDeleteNodeComponent,
        WorkflowJoinComponent,
        WorkflowJoinTriggerSrcComponent,
        WorkflowNodeContextComponent,
        WorkflowNodeFormComponent,
        WorkflowNodeConditionsComponent,
        WorkflowNodeHookFormComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeHookDetailsComponent,
        WorkflowTriggerComponent,
        WorkflowTriggerJoinComponent,
        WorkflowNodeConditionFormComponent,
        WorkflowNodeConditionListComponent,
        ZoneComponent,
        ZoneContentComponent,
        UsageWorkflowsComponent,
        UsageApplicationsComponent,
        UsagePipelinesComponent,
        UsageEnvironmentsComponent,
        UsageComponent,
        DiffComponent,
        SpanColoredComponent,
        VCSStrategyComponent,
        FavoriteCardsComponent
    ],
    entryComponents: [SpanColoredComponent],
    providers: [
        DurationService,
        PermissionService,
        BroadcastLevelService,
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
        CommitListComponent,
        CommonModule,
        ClipboardModule,
        CutPipe,
        DeleteButtonComponent,
        UploadButtonComponent,
        DragulaModule,
        ForMapPipe,
        FormsModule,
        GroupFormComponent,
        HistoryComponent,
        KeysPipe,
        KeysFormComponent,
        KeysListComponent,
        MarkdownModule,
        MomentModule,
        NgForNumber,
        TokenListComponent,
        NgSemanticModule,
        NgxAutoScroll,
        ParameterDescriptionComponent,
        ParameterListComponent,
        ParameterFormComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        PermissionEnvironmentFormComponent,
        PrerequisiteComponent,
        PrerequisitesFormComponent,
        PrerequisiteComponent,
        PrerequisitesFormComponent,
        PipelineLaunchModalComponent,
        ReactiveFormsModule,
        RepoManagerFormComponent,
        StatusIconComponent,
        SuiModule,
        TranslateModule,
        TruncatePipe,
        VariableComponent,
        VariableFormComponent,
        VariableValueComponent,
        WarningTabComponent,
        WarningModalComponent,
        DeleteModalComponent,
        WorkflowNodeComponent,
        WorkflowNodeAddWizardComponent,
        WorkflowDeleteJoinComponent,
        WorkflowDeleteNodeComponent,
        WorkflowJoinComponent,
        WorkflowJoinTriggerSrcComponent,
        WorkflowNodeContextComponent,
        WorkflowNodeFormComponent,
        WorkflowNodeConditionsComponent,
        WorkflowNodeRunParamComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeHookDetailsComponent,
        WorkflowNodeHookFormComponent,
        WorkflowTriggerComponent,
        WorkflowTriggerJoinComponent,
        ZoneComponent,
        ZoneContentComponent,
        UsageWorkflowsComponent,
        UsageApplicationsComponent,
        UsagePipelinesComponent,
        UsageEnvironmentsComponent,
        UsageComponent,
        DiffComponent,
        SpanColoredComponent,
        VCSStrategyComponent,
        FavoriteCardsComponent
    ]
})
export class SharedModule {
}
