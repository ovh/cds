import { CommonModule } from '@angular/common';
import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { NguiAutoCompleteModule } from '@ngui/auto-complete';
import { TranslateModule } from '@ngx-translate/core';
import { SuiModule } from '@richardlt/ng2-semantic-ui';
import { NgxChartsModule } from '@swimlane/ngx-charts';
import { ConditionsComponent } from 'app/shared/conditions/conditions.component';
import { GroupFormComponent } from 'app/shared/group/form/group.form.component';
import { WorkflowHookMenuEditComponent } from 'app/shared/workflow/menu/edit-hook/menu.edit.hook.component';
import { WorkflowWizardNodeConditionComponent } from 'app/shared/workflow/wizard/conditions/wizard.conditions.component';
import { WorkflowWizardOutgoingHookComponent } from 'app/shared/workflow/wizard/outgoinghook/wizard.outgoinghook.component';
import { NgSemanticModule } from 'ng-semantic/ng-semantic';
import { CodemirrorModule } from 'ng2-codemirror-typescript/Codemirror';
import { DragulaModule } from 'ng2-dragula';
import { NgxAutoScroll, NgxAutoScrollModule } from 'ngx-auto-scroll';
import { ClipboardModule } from 'ngx-clipboard';
import { InfiniteScrollModule } from 'ngx-infinite-scroll';
import { MarkdownModule } from 'ngx-markdown';
import { MomentModule } from 'ngx-moment';
import { ActionComponent } from './action/action.component';
import { ActionStepFormComponent } from './action/step/form/step.form.component';
import { ActionStepComponent } from './action/step/step.component';
import { AuditListComponent } from './audit/list/audit.list.component';
import { BreadcrumbComponent } from './breadcrumb/breadcrumb.component';
import { ConfirmButtonComponent } from './button/confirm/confirm.button';
import { DeleteButtonComponent } from './button/delete/delete.button';
import { UploadButtonComponent } from './button/upload/upload.button.component';
import { ChartComponentComponent } from './chart/chart.component';
import { CommitListComponent } from './commit/commit.list.component';
import { DiffItemComponent } from './diff/item/diff.item.component';
import { DiffListComponent } from './diff/list/diff.list.component';
import { DurationService } from './duration/duration.service';
import { FavoriteCardsComponent } from './favorite-cards/favorite-cards.component';
import { KeysFormComponent } from './keys/form/keys.form.component';
import { KeysListComponent } from './keys/list/keys.list.component';
import { LabelsEditComponent } from './labels/edit/labels.edit.component';
import { ConfirmModalComponent } from './modal/confirm/confirm.component';
import { DeleteModalComponent } from './modal/delete/delete.component';
import { WarningModalComponent } from './modal/warning/warning.component';
import { ParameterDescriptionComponent } from './parameter/description-popup/description.popup.component';
import { ParameterFormComponent } from './parameter/form/parameter.form';
import { ParameterListComponent } from './parameter/list/parameter.component';
import { ParameterValueComponent } from './parameter/value/parameter.value.component';
import { PermissionFormComponent } from './permission/form/permission.form.component';
import { PermissionListComponent } from './permission/list/permission.list.component';
import { PermissionService } from './permission/permission.service';
import { CutPipe } from './pipes/cut.pipe';
import { KeysPipe } from './pipes/keys.pipe';
import { ForMapPipe } from './pipes/map.pipe';
import { NgForNumber } from './pipes/ngfor.number.pipe';
import { SafeHtmlPipe } from './pipes/safeHtml.pipe';
import { TruncatePipe } from './pipes/truncate.pipe';
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
import { WorkflowWNodeMenuEditComponent } from './workflow/menu/edit-node/menu.edit.node.component';
import { WorkflowDeleteNodeComponent } from './workflow/modal/delete/workflow.node.delete.component';
import { WorkflowHookModalComponent } from './workflow/modal/hook-add/hook.modal.component';
import { WorkflowTriggerComponent } from './workflow/modal/node-add/workflow.trigger.component';
import { WorkflowNodeEditModalComponent } from './workflow/modal/node-edit/node.edit.modal.component';
import { WorkflowSaveAsCodeComponent } from './workflow/modal/save-as-code/save.as.code.component';
import { WorkflowNodeHookDetailsComponent } from './workflow/node/hook/details/hook.details.component';
import { WorkflowNodeRunParamComponent } from './workflow/node/run/node.run.param.component';
import { WorkflowSidebarHookComponent } from './workflow/sidebar/edit-hook/workflow.sidebar.hook.component';
import { WorkflowSidebarRunHookComponent } from './workflow/sidebar/run-hook/workflow.sidebar.run.hook.component';
import { WorkflowSidebarRunListComponent } from './workflow/sidebar/run-list/workflow.sidebar.run.component';
import { ActionStepSummaryComponent } from './workflow/sidebar/run-node/stage/job/action/action.summary.component';
import { JobStepSummaryComponent } from './workflow/sidebar/run-node/stage/job/job.summary.component';
import { StageStepSummaryComponent } from './workflow/sidebar/run-node/stage/stage.summary.component';
import { WorkflowSidebarRunNodeComponent } from './workflow/sidebar/run-node/workflow.sidebar.run.node.component';
import { WorkflowWizardNodeContextComponent } from './workflow/wizard/context/wizard.context.component';
import { WorkflowNodeHookFormComponent } from './workflow/wizard/hook/hook.form.component';
import { WorkflowWizardNodeInputComponent } from './workflow/wizard/input/wizard.input.component';
import { WorkflowNodeAddWizardComponent } from './workflow/wizard/node-add/node.wizard.component';
import { WorkflowWNodeForkComponent } from './workflow/wnode/fork/node.fork.component';
import { WorkflowNodeHookComponent } from './workflow/wnode/hook/hook.component';
import { WorkflowWNodeJoinComponent } from './workflow/wnode/join/node.join.component';
import { WorkflowWNodeOutGoingHookComponent } from './workflow/wnode/outgoinghook/node.outgoinghook.component';
import { WorkflowWNodePipelineComponent } from './workflow/wnode/pipeline/wnode.pipeline.component';
import { WorkflowWNodeComponent } from './workflow/wnode/wnode.component';
import { ZoneContentComponent } from './zone/zone-content/content.component';
import { ZoneComponent } from './zone/zone.component';

@NgModule({
    imports: [
        CommonModule,
        ClipboardModule,
        NgSemanticModule,
        FormsModule,
        TranslateModule,
        DragulaModule.forRoot(),
        MomentModule,
        CodemirrorModule,
        ReactiveFormsModule,
        MarkdownModule.forRoot(),
        NguiAutoCompleteModule,
        RouterModule,
        SuiModule,
        NgxAutoScrollModule,
        InfiniteScrollModule,
        NgxChartsModule
    ],
    declarations: [
        ActionComponent,
        ActionStepComponent,
        ActionStepFormComponent,
        ActionStepSummaryComponent,
        AuditListComponent,
        BreadcrumbComponent,
        ChartComponentComponent,
        CommitListComponent,
        ConditionsComponent,
        ConfirmButtonComponent,
        ConfirmModalComponent,
        CutPipe,
        DataTableComponent,
        DeleteButtonComponent,
        DeleteModalComponent,
        DiffItemComponent,
        DiffListComponent,
        FavoriteCardsComponent,
        ForMapPipe,
        GroupFormComponent,
        JobStepSummaryComponent,
        KeysFormComponent,
        KeysListComponent,
        KeysPipe,
        LabelsEditComponent,
        NgForNumber,
        ParameterDescriptionComponent,
        ParameterFormComponent,
        ParameterListComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        ProjectBreadcrumbComponent,
        RequirementsFormComponent,
        RequirementsListComponent,
        RepoManagerFormComponent,
        SafeHtmlPipe,
        SelectorPipe,
        SelectPipe,
        ScrollviewComponent,
        StageStepSummaryComponent,
        StatusIconComponent,
        TabsComponent,
        TokenListComponent,
        TruncatePipe,
        UploadButtonComponent,
        UsageApplicationsComponent,
        UsageComponent,
        UsageEnvironmentsComponent,
        UsagePipelinesComponent,
        UsageWorkflowsComponent,
        VariableAuditComponent,
        VariableComponent,
        VariableDiffComponent,
        VariableFormComponent,
        VariableValueComponent,
        VCSStrategyComponent,
        VulnerabilitiesComponent,
        VulnerabilitiesListComponent,
        WarningMarkComponent,
        WarningMarkListComponent,
        WarningModalComponent,
        WarningTabComponent,
        WorkflowDeleteNodeComponent,
        WorkflowHookMenuEditComponent,
        WorkflowHookModalComponent,
        WorkflowNodeAddWizardComponent,
        WorkflowNodeEditModalComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeHookDetailsComponent,
        WorkflowNodeHookFormComponent,
        WorkflowNodeRunParamComponent,
        WorkflowSaveAsCodeComponent,
        WorkflowSidebarHookComponent,
        WorkflowSidebarRunListComponent,
        WorkflowSidebarRunHookComponent,
        WorkflowSidebarRunNodeComponent,
        WorkflowTemplateApplyFormComponent,
        WorkflowTemplateApplyModalComponent,
        WorkflowTemplateBulkModalComponent,
        WorkflowTemplateParamFormComponent,
        WorkflowTriggerComponent,
        WorkflowWizardNodeConditionComponent,
        WorkflowWizardNodeContextComponent,
        WorkflowWizardNodeInputComponent,
        WorkflowWizardOutgoingHookComponent,
        WorkflowWNodeComponent,
        WorkflowWNodeForkComponent,
        WorkflowWNodeJoinComponent,
        WorkflowWNodeMenuEditComponent,
        WorkflowWNodeOutGoingHookComponent,
        WorkflowWNodePipelineComponent,
        ZoneComponent,
        ZoneContentComponent
    ],
    entryComponents: [],
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
        AuditListComponent,
        ActionStepComponent,
        ActionStepFormComponent,
        BreadcrumbComponent,
        ProjectBreadcrumbComponent,
        ChartComponentComponent,
        CodemirrorModule,
        CommitListComponent,
        CommonModule,
        ClipboardModule,
        CutPipe,
        GroupFormComponent,
        DeleteButtonComponent,
        ConfirmButtonComponent,
        UploadButtonComponent,
        DragulaModule,
        ForMapPipe,
        FormsModule,
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
        ConditionsComponent,
        ParameterDescriptionComponent,
        ParameterListComponent,
        ParameterFormComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        ReactiveFormsModule,
        RepoManagerFormComponent,
        RequirementsFormComponent,
        RequirementsListComponent,
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
        ConfirmModalComponent,
        LabelsEditComponent,
        WorkflowWNodeComponent,
        WorkflowSidebarHookComponent,
        WorkflowSidebarRunListComponent,
        WorkflowSidebarRunNodeComponent,
        WorkflowSidebarRunHookComponent,
        WorkflowSaveAsCodeComponent,
        WorkflowWNodeMenuEditComponent,
        WorkflowNodeAddWizardComponent,
        WorkflowDeleteNodeComponent,
        WorkflowNodeRunParamComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeHookDetailsComponent,
        WorkflowNodeHookFormComponent,
        WorkflowTriggerComponent,
        WorkflowNodeEditModalComponent,
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
