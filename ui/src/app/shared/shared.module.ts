import { CommonModule } from '@angular/common';
import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { NguiAutoCompleteModule } from '@ngui/auto-complete';
import { TranslateModule } from '@ngx-translate/core';
import { NgxChartsModule } from '@swimlane/ngx-charts';
import { MomentModule } from 'angular2-moment';
import { NgSemanticModule } from 'ng-semantic/ng-semantic';
import { CodemirrorModule } from 'ng2-codemirror-typescript/Codemirror';
import { DragulaModule } from 'ng2-dragula/ng2-dragula';
import { SuiModule } from 'ng2-semantic-ui';
import { NgxAutoScroll, NgxAutoScrollModule } from 'ngx-auto-scroll';
import { ClipboardModule } from 'ngx-clipboard';
import { InfiniteScrollModule } from 'ngx-infinite-scroll';
import { MarkdownModule } from 'ngx-markdown';
import { ActionComponent } from './action/action.component';
import { ActionStepFormComponent } from './action/step/form/step.form.component';
import { ActionStepComponent } from './action/step/step.component';
import { AuditListComponent } from './audit/list/audit.list.component';
import { BreadcrumbComponent } from './breadcrumb/breadcrumb.component';
import { BroadcastLevelService } from './broadcast/broadcast.level.service';
import { ConfirmButtonComponent } from './button/confirm/confirm.button';
import { DeleteButtonComponent } from './button/delete/delete.button';
import { UploadButtonComponent } from './button/upload/upload.button.component';
import { ChartComponentComponent } from './chart/chart.component';
import { CommitListComponent } from './commit/commit.list.component';
import { DiffItemComponent } from './diff/item/diff.item.component';
import { DiffListComponent } from './diff/list/diff.list.component';
import { DurationService } from './duration/duration.service';
import { FavoriteCardsComponent } from './favorite-cards/favorite-cards.component';
import { GroupFormComponent } from './group/form/group.form.component';
import { KeysFormComponent } from './keys/form/keys.form.component';
import { KeysListComponent } from './keys/list/keys.list.component';
import { LabelsEditComponent } from './labels/edit/labels.edit.component';
import { DeleteModalComponent } from './modal/delete/delete.component';
import { WarningModalComponent } from './modal/warning/warning.component';
import { ParameterDescriptionComponent } from './parameter/description-popup/description.popup.component';
import { ParameterFormComponent } from './parameter/form/parameter.form';
import { ParameterListComponent } from './parameter/list/parameter.component';
import { ParameterValueComponent } from './parameter/value/parameter.value.component';
import { PermissionEnvironmentFormComponent } from './permission/environment/form/permission.env.form.component';
import { PermissionFormComponent } from './permission/form/permission.form.component';
import { PermissionListComponent } from './permission/list/permission.list.component';
import { PermissionService } from './permission/permission.service';
import { CutPipe } from './pipes/cut.pipe';
import { KeysPipe } from './pipes/keys.pipe';
import { ForMapPipe } from './pipes/map.pipe';
import { NgForNumber } from './pipes/ngfor.number.pipe';
import { SafeHtmlPipe } from './pipes/safeHtml.pipe';
import { TruncatePipe } from './pipes/truncate.pipe';
import { PrerequisitesFormComponent } from './prerequisites/form/prerequisites.form.component';
import { PrerequisiteComponent } from './prerequisites/list/prerequisites.component';
import { ProjectBreadcrumbComponent } from './project-breadcrumb/project-breadcrumb.component';
import { RepoManagerFormComponent } from './repomanager/from/repomanager.form.component';
import { RequirementsFormComponent } from './requirements/form/requirements.form.component';
import { RequirementsListComponent } from './requirements/list/requirements.list.component';
import { ScrollviewComponent } from './scrollview/scrollview.component';
import { SharedService } from './shared.service';
import { StatusIconComponent } from './status/status.component';
import { DataTableComponent, SelectorPipe, SelectPipe } from './table/data-table.component';
import { TabsComponent } from './tabs/tabs.component';
import { ToastService } from './toast/ToastService';
import { TokenListComponent } from './token/list/token.list.component';
import { UsageApplicationsComponent } from './usage/applications/usage.applications.component';
import { UsageEnvironmentsComponent } from './usage/environments/usage.environments.component';
import { UsagePipelinesComponent } from './usage/pipelines/usage.pipelines.component';
import { UsageComponent } from './usage/usage.component';
import { UsageWorkflowsComponent } from './usage/workflows/usage.workflows.component';
import { VariableAuditComponent } from './variable/audit/audit.component';
import { VariableDiffComponent } from './variable/diff/variable.diff.component';
import { VariableFormComponent } from './variable/form/variable.form';
import { VariableComponent } from './variable/list/variable.component';
import { VariableValueComponent } from './variable/value/variable.value.component';
import { VCSStrategyComponent } from './vcs/vcs.strategy.component';
import { VulnerabilitiesListComponent } from './vulnerability/list/vulnerability.list.component';
import { VulnerabilitiesComponent } from './vulnerability/vulnerabilities.component';
import { WarningMarkListComponent } from './warning/mark-list/warning.mark.list.component';
import { WarningMarkComponent } from './warning/mark-single/warning.mark.component';
import { WarningTabComponent } from './warning/tab/warning.tab.component';
import { WorkflowTemplateApplyFormComponent } from './workflow-template/apply-form/workflow-template.apply-form.component';
import { WorkflowTemplateApplyModalComponent } from './workflow-template/apply-modal/workflow-template.apply-modal.component';
import { WorkflowTemplateBulkModalComponent } from './workflow-template/bulk-modal/workflow-template.bulk-modal.component';
import { WorkflowTemplateParamFormComponent } from './workflow-template/param-form/workflow-template.param-form.component';
import { WorkflowNodeConditionFormComponent } from './workflow/modal/conditions/condition-form/condition.component';
import { WorkflowNodeConditionListComponent } from './workflow/modal/conditions/condition-list/condition.list.component';
import { WorkflowNodeConditionsComponent } from './workflow/modal/conditions/node.conditions.component';
import { WorkflowNodeContextComponent } from './workflow/modal/context/workflow.node.context.component';
import { WorkflowDeleteNodeComponent } from './workflow/modal/delete/workflow.node.delete.component';
import { WorkflowHookModalComponent } from './workflow/modal/hook-modal/hook.modal.component';
import { WorkflowNodeOutGoingHookEditComponent } from './workflow/modal/outgoinghook-edit/outgoinghook.edit.component';
import { WorkflowSaveAsCodeComponent } from './workflow/modal/save-as-code/save.as.code.component';
import { WorkflowTriggerComponent } from './workflow/modal/trigger/workflow.trigger.component';
import { WorkflowNodeHookDetailsComponent } from './workflow/node/hook/details/hook.details.component';
import { WorkflowNodeHookFormComponent } from './workflow/node/hook/form/hook.form.component';
import { WorkflowNodeOutGoingHookFormComponent } from './workflow/node/outgoinghook-form/outgoinghook.form.component';
import { WorkflowNodeFormComponent } from './workflow/node/pipeline-form/workflow.node.form.component';
import { WorkflowNodeRunParamComponent } from './workflow/node/run/node.run.param.component';
import { WorkflowNodeAddWizardComponent } from './workflow/node/wizard/node.wizard.component';
import { WorkflowSidebarHookComponent } from './workflow/sidebar/edit-hook/workflow.sidebar.hook.component';
import { WorkflowWNodeSidebarEditComponent } from './workflow/sidebar/edit-node/sidebar.edit.component';
import { WorkflowSidebarRunHookComponent } from './workflow/sidebar/run-hook/workflow.sidebar.run.hook.component';
import { WorkflowSidebarRunListComponent } from './workflow/sidebar/run-list/workflow.sidebar.run.component';
import { ActionStepSummaryComponent } from './workflow/sidebar/run-node/stage/job/action/action.summary.component';
import { JobStepSummaryComponent } from './workflow/sidebar/run-node/stage/job/job.summary.component';
import { StageStepSummaryComponent } from './workflow/sidebar/run-node/stage/stage.summary.component';
import { WorkflowSidebarRunNodeComponent } from './workflow/sidebar/run-node/workflow.sidebar.run.node.component';
import { WorkflowWNodeForkComponent } from './workflow/wnode/fork/node.fork.component';
import { WorkflowNodeHookComponent } from './workflow/wnode/hook/hook.component';
import { WorkflowWNodeJoinComponent } from './workflow/wnode/join/node.join.component';
import { WorkflowWNodeOutGoingHookComponent } from './workflow/wnode/outgoinghook/node.outgoinghook.component';
import { WorkflowWNodePipelineComponent } from './workflow/wnode/pipeline/wnode.pipeline.component';
import { WorkflowWNodeComponent } from './workflow/wnode/wnode.component';
import { ZoneContentComponent } from './zone/zone-content/content.component';
import { ZoneComponent } from './zone/zone.component';

@NgModule({
    imports: [CommonModule, ClipboardModule, NgSemanticModule, FormsModule, TranslateModule, DragulaModule, MomentModule,
        CodemirrorModule, ReactiveFormsModule, MarkdownModule.forRoot(), NguiAutoCompleteModule, RouterModule,
        SuiModule, NgxAutoScrollModule, InfiniteScrollModule, NgxChartsModule],
    declarations: [
        ActionComponent,
        ActionStepComponent,
        ActionStepFormComponent,
        AuditListComponent,
        BreadcrumbComponent,
        ProjectBreadcrumbComponent,
        ChartComponentComponent,
        CommitListComponent,
        CutPipe,
        DeleteButtonComponent,
        ConfirmButtonComponent,
        UploadButtonComponent,
        ForMapPipe,
        GroupFormComponent,
        KeysPipe,
        KeysFormComponent,
        KeysListComponent,
        NgForNumber,
        TokenListComponent,
        ParameterDescriptionComponent,
        ParameterListComponent,
        ParameterFormComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        PermissionEnvironmentFormComponent,
        PrerequisiteComponent,
        PrerequisitesFormComponent,
        RequirementsListComponent,
        RequirementsFormComponent,
        RepoManagerFormComponent,
        StatusIconComponent,
        TruncatePipe,
        SafeHtmlPipe,
        VariableComponent,
        VariableAuditComponent,
        VariableDiffComponent,
        VariableFormComponent,
        VariableValueComponent,
        VulnerabilitiesComponent,
        VulnerabilitiesListComponent,
        WarningModalComponent,
        DeleteModalComponent,
        LabelsEditComponent,
        WarningTabComponent,
        WarningMarkComponent,
        WarningMarkListComponent,

        WorkflowWNodeComponent,
        WorkflowWNodeForkComponent,
        WorkflowWNodeJoinComponent,
        WorkflowWNodeOutGoingHookComponent,
        WorkflowWNodePipelineComponent,
        WorkflowWNodeSidebarEditComponent,
        WorkflowNodeOutGoingHookFormComponent,
        WorkflowNodeOutGoingHookEditComponent,
        WorkflowHookModalComponent,
        WorkflowSidebarHookComponent,
        WorkflowSidebarRunListComponent,
        WorkflowSidebarRunNodeComponent,
        StageStepSummaryComponent,
        JobStepSummaryComponent,
        ActionStepSummaryComponent,
        WorkflowSidebarRunHookComponent,
        WorkflowSaveAsCodeComponent,

        WorkflowNodeAddWizardComponent,
        WorkflowNodeRunParamComponent,
        WorkflowDeleteNodeComponent,
        WorkflowNodeContextComponent,
        WorkflowNodeFormComponent,
        WorkflowNodeConditionsComponent,
        WorkflowNodeHookFormComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeHookDetailsComponent,
        WorkflowTriggerComponent,
        WorkflowNodeConditionFormComponent,
        WorkflowNodeConditionListComponent,
        ZoneComponent,
        ZoneContentComponent,
        UsageWorkflowsComponent,
        UsageApplicationsComponent,
        UsagePipelinesComponent,
        UsageEnvironmentsComponent,
        UsageComponent,
        DiffItemComponent,
        DiffListComponent,
        VCSStrategyComponent,
        FavoriteCardsComponent,
        SelectorPipe,
        SelectPipe,
        DataTableComponent,
        WorkflowTemplateApplyFormComponent,
        WorkflowTemplateApplyModalComponent,
        WorkflowTemplateBulkModalComponent,
        WorkflowTemplateParamFormComponent,
        TabsComponent,
        ScrollviewComponent
    ],
    entryComponents: [],
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
        AuditListComponent,
        BreadcrumbComponent,
        ProjectBreadcrumbComponent,
        ChartComponentComponent,
        CodemirrorModule,
        CommitListComponent,
        CommonModule,
        ClipboardModule,
        CutPipe,
        DeleteButtonComponent,
        ConfirmButtonComponent,
        UploadButtonComponent,
        DragulaModule,
        ForMapPipe,
        FormsModule,
        GroupFormComponent,
        KeysPipe,
        KeysFormComponent,
        KeysListComponent,
        InfiniteScrollModule,
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
        ReactiveFormsModule,
        RepoManagerFormComponent,
        StatusIconComponent,
        SuiModule,
        TranslateModule,
        TruncatePipe,
        SafeHtmlPipe,
        VariableComponent,
        VariableFormComponent,
        VariableValueComponent,
        VulnerabilitiesComponent,
        VulnerabilitiesListComponent,
        WarningTabComponent,
        WarningMarkComponent,
        WarningMarkListComponent,
        WarningModalComponent,
        DeleteModalComponent,
        LabelsEditComponent,

        WorkflowWNodeComponent,
        WorkflowWNodeSidebarEditComponent,
        WorkflowSidebarHookComponent,
        WorkflowSidebarRunListComponent,
        WorkflowSidebarRunNodeComponent,
        WorkflowSidebarRunHookComponent,
        WorkflowSaveAsCodeComponent,

        WorkflowNodeAddWizardComponent,
        WorkflowDeleteNodeComponent,
        WorkflowNodeContextComponent,
        WorkflowNodeFormComponent,
        WorkflowNodeConditionsComponent,
        WorkflowNodeRunParamComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeHookDetailsComponent,
        WorkflowNodeHookFormComponent,
        WorkflowTriggerComponent,
        ZoneComponent,
        ZoneContentComponent,
        UsageWorkflowsComponent,
        UsageApplicationsComponent,
        UsagePipelinesComponent,
        UsageEnvironmentsComponent,
        UsageComponent,
        DiffItemComponent,
        DiffListComponent,
        VCSStrategyComponent,
        FavoriteCardsComponent,
        SelectorPipe,
        SelectPipe,
        DataTableComponent,
        WorkflowTemplateApplyFormComponent,
        WorkflowTemplateApplyModalComponent,
        WorkflowTemplateBulkModalComponent,
        WorkflowTemplateParamFormComponent,
        TabsComponent,
        ScrollviewComponent
    ]
})
export class SharedModule {
}
