import { CommonModule } from '@angular/common';
import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { NguiAutoCompleteComponent, NguiAutoCompleteModule } from '@ngui/auto-complete';
import { TranslateModule } from '@ngx-translate/core';
import { SuiModule } from '@richardlt/ng2-semantic-ui';
import { NgxChartsModule } from '@swimlane/ngx-charts';
import { AuthenticationGuard } from 'app/guard/authentication.guard';
import { FeatureGuard } from 'app/guard/feature.guard';
import { NoAuthenticationGuard } from 'app/guard/no-authentication.guard';
import { AsCodeEventComponent } from 'app/shared/ascode/events/ascode.event.component';
import { AsCodeSaveFormComponent } from 'app/shared/ascode/save-form/ascode.save-form.component';
import { AsCodeSaveModalComponent } from 'app/shared/ascode/save-modal/ascode.save-modal.component';
import { ConditionsComponent } from 'app/shared/conditions/conditions.component';
import { GroupFormComponent } from 'app/shared/group/form/group.form.component';
import { AutoFocusInputComponent } from 'app/shared/input/autofocus/autofocus.input.component';
import { CallbackPipe } from 'app/shared/pipes/callback.pipe';
import { SelectFilterComponent } from 'app/shared/select/select.component';
import { WorkflowHookMenuEditComponent } from 'app/shared/workflow/menu/edit-hook/menu.edit.hook.component';
import { WorkflowWizardNodeConditionComponent } from 'app/shared/workflow/wizard/conditions/wizard.conditions.component';
import { WorkflowWizardOutgoingHookComponent } from 'app/shared/workflow/wizard/outgoinghook/wizard.outgoinghook.component';
import { WorkflowRunJobVariableComponent } from 'app/views/workflow/run/node/pipeline/variables/job.variables.component';
import { WorkflowRunJobComponent } from 'app/views/workflow/run/node/pipeline/workflow-run-job/workflow-run-job.component';
import { NgSemanticModule } from 'ng-semantic/ng-semantic';
import { CodemirrorModule } from 'ng2-codemirror-typescript/Codemirror';
import { DragulaModule } from 'ng2-dragula-sgu';
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
import { FavoriteCardsComponent } from './favorite-cards/favorite-cards.component';
import { KeysFormComponent } from './keys/form/keys.form.component';
import { KeysListComponent } from './keys/list/keys.list.component';
import { LabelsEditComponent } from './labels/edit/labels.edit.component';
import { MenuComponent } from './menu/menu.component';
import { ConfirmModalComponent } from './modal/confirm/confirm.component';
import { DeleteModalComponent } from './modal/delete/delete.component';
import { WarningModalComponent } from './modal/warning/warning.component';
import { ParameterFormComponent } from './parameter/form/parameter.form';
import { ParameterListComponent } from './parameter/list/parameter.component';
import { ParameterValueComponent } from './parameter/value/parameter.value.component';
import { PermissionFormComponent } from './permission/form/permission.form.component';
import { PermissionListComponent } from './permission/list/permission.list.component';
import { PermissionService } from './permission/permission.service';
import { AnsiPipe } from './pipes/ansi.pipe';
import { CutPipe } from './pipes/cut.pipe';
import { DurationMsPipe } from './pipes/duration.pipe';
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
import { PaginationComponent } from './table/pagination.component';
import { TabsComponent } from './tabs/tabs.component';
import { ToastHTTPErrorComponent } from './toast/toast-http-error.component';
import { ToastService } from './toast/ToastService';
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
import { WorkflowTemplateApplyFormComponent } from './workflow-template/apply-form/workflow-template.apply-form.component';
import { WorkflowTemplateApplyModalComponent } from './workflow-template/apply-modal/workflow-template.apply-modal.component';
import { WorkflowTemplateBulkModalComponent } from './workflow-template/bulk-modal/workflow-template.bulk-modal.component';
import { WorkflowTemplateParamFormComponent } from './workflow-template/param-form/workflow-template.param-form.component';
import { WorkflowWNodeMenuEditComponent } from './workflow/menu/edit-node/menu.edit.node.component';
import { WorkflowDeleteNodeComponent } from './workflow/modal/delete/workflow.node.delete.component';
import { WorkflowHookModalComponent } from './workflow/modal/hook-add/hook.modal.component';
import { WorkflowTriggerComponent } from './workflow/modal/node-add/workflow.trigger.component';
import { WorkflowNodeEditModalComponent } from './workflow/modal/node-edit/node.edit.modal.component';
import { WorkflowNodeHookDetailsComponent } from './workflow/node/hook/details/hook.details.component';
import { WorkflowNodeRunParamComponent } from './workflow/node/run/node.run.param.component';
import { WorkflowSidebarRunListComponent } from './workflow/sidebar/run-list/workflow.sidebar.run.component';
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
        AnsiPipe,
        AsCodeEventComponent,
        AsCodeSaveFormComponent,
        AsCodeSaveModalComponent,
        AuditListComponent,
        AutoFocusInputComponent,
        BreadcrumbComponent,
        CallbackPipe,
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
        DurationMsPipe,
        FavoriteCardsComponent,
        ForMapPipe,
        GroupFormComponent,
        KeysFormComponent,
        KeysListComponent,
        KeysPipe,
        LabelsEditComponent,
        MenuComponent,
        NgForNumber,
        PaginationComponent,
        ParameterFormComponent,
        ParameterListComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        ProjectBreadcrumbComponent,
        RepoManagerFormComponent,
        RequirementsFormComponent,
        RequirementsListComponent,
        SafeHtmlPipe,
        ScrollviewComponent,
        SelectFilterComponent,
        SelectorPipe,
        SelectPipe,
        StatusIconComponent,
        TabsComponent,
        ToastHTTPErrorComponent,
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
        WarningModalComponent,
        WorkflowDeleteNodeComponent,
        WorkflowHookMenuEditComponent,
        WorkflowHookModalComponent,
        WorkflowNodeAddWizardComponent,
        WorkflowNodeEditModalComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeHookDetailsComponent,
        WorkflowNodeHookFormComponent,
        WorkflowNodeRunParamComponent,
        WorkflowRunJobComponent,
        WorkflowRunJobVariableComponent,
        WorkflowSidebarRunListComponent,
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
    entryComponents: [
        NguiAutoCompleteComponent,
        ToastHTTPErrorComponent
    ],
    providers: [
        PermissionService,
        SharedService,
        ToastService,
        AuthenticationGuard,
        NoAuthenticationGuard,
        FeatureGuard
    ],
    schemas: [
        CUSTOM_ELEMENTS_SCHEMA
    ],
    exports: [
        ActionComponent,
        ActionStepComponent,
        ActionStepFormComponent,
        AnsiPipe,
        AsCodeEventComponent,
        AsCodeSaveFormComponent,
        AsCodeSaveModalComponent,
        AuditListComponent,
        AutoFocusInputComponent,
        BreadcrumbComponent,
        CallbackPipe,
        ChartComponentComponent,
        ClipboardModule,
        CodemirrorModule,
        CommitListComponent,
        CommonModule,
        ConditionsComponent,
        ConfirmButtonComponent,
        ConfirmModalComponent,
        CutPipe,
        DataTableComponent,
        DeleteButtonComponent,
        DeleteModalComponent,
        DiffItemComponent,
        DiffListComponent,
        DragulaModule,
        DurationMsPipe,
        FavoriteCardsComponent,
        ForMapPipe,
        FormsModule,
        GroupFormComponent,
        InfiniteScrollModule,
        KeysFormComponent,
        KeysListComponent,
        KeysPipe,
        LabelsEditComponent,
        MarkdownModule,
        MenuComponent,
        MomentModule,
        NgForNumber,
        NgSemanticModule,
        NgxAutoScroll,
        PaginationComponent,
        ParameterFormComponent,
        ParameterListComponent,
        ParameterValueComponent,
        PermissionFormComponent,
        PermissionListComponent,
        ProjectBreadcrumbComponent,
        ReactiveFormsModule,
        RepoManagerFormComponent,
        RequirementsFormComponent,
        RequirementsListComponent,
        SafeHtmlPipe,
        ScrollviewComponent,
        SelectorPipe,
        SelectPipe,
        StatusIconComponent,
        SuiModule,
        TabsComponent,
        ToastHTTPErrorComponent,
        TranslateModule,
        TruncatePipe,
        UploadButtonComponent,
        UsageApplicationsComponent,
        UsageComponent,
        UsageEnvironmentsComponent,
        UsagePipelinesComponent,
        UsageWorkflowsComponent,
        VariableComponent,
        VariableFormComponent,
        VariableValueComponent,
        VCSStrategyComponent,
        VulnerabilitiesComponent,
        VulnerabilitiesListComponent,
        WarningModalComponent,
        WorkflowDeleteNodeComponent,
        WorkflowNodeAddWizardComponent,
        WorkflowNodeEditModalComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeHookDetailsComponent,
        WorkflowNodeHookFormComponent,
        WorkflowNodeRunParamComponent,
        WorkflowRunJobComponent,
        WorkflowRunJobVariableComponent,
        WorkflowSidebarRunListComponent,
        WorkflowTemplateApplyFormComponent,
        WorkflowTemplateApplyModalComponent,
        WorkflowTemplateBulkModalComponent,
        WorkflowTemplateParamFormComponent,
        WorkflowTriggerComponent,
        WorkflowWNodeComponent,
        WorkflowWNodeMenuEditComponent,
        ZoneComponent,
        ZoneContentComponent
    ]
})
export class SharedModule {
}
