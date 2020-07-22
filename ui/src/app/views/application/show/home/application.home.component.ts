import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Subscription } from 'rxjs';
import { filter } from 'rxjs/operators';
import { Application, Overview, Severity } from '../../../../model/application.model';
import { ChartData, ChartSeries, GraphConfiguration, GraphType } from '../../../../model/graph.model';
import { Metric } from '../../../../model/metric.model';
import { Tests } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';
import { FetchApplicationOverview } from '../../../../store/applications.action';
import { ApplicationsState } from '../../../../store/applications.state';

@Component({
    selector: 'app-home',
    templateUrl: './application.home.html',
    styleUrls: ['./application.home.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ApplicationHomeComponent implements OnInit, OnDestroy {
    @Input() project: Project;
    @Input() application: Application;

    dashboards: Array<GraphConfiguration>;
    overview: Overview;
    overviewSubscription: Subscription;

    constructor(
        private _translate: TranslateService,
        private store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.store.dispatch(new FetchApplicationOverview({ projectKey: this.project.key, applicationName: this.application.name }));
        this.overviewSubscription = this.store.select(ApplicationsState.selectOverview())
            .pipe(filter((o) => o != null))
            .subscribe((o: Overview) => {
                this.overview = o;
                this.renderGraph();
                this._cd.markForCheck();
            });
    }

    renderGraph(): void {
        this.dashboards = new Array<GraphConfiguration>();
        if (this.overview?.graphs?.length > 0) {
            this.overview.graphs.forEach(g => {
                if (g.datas && g.datas.length) {
                    switch (g.type) {
                        case 'Vulnerability':
                            this.createVulnDashboard(g.datas);
                            break;
                        case 'UnitTest':
                            this.createUnitTestDashboard(g.datas);
                            break;
                        case 'Coverage':
                            this.createCoverageDashboard(g.datas);
                            break;
                    }
                }
            });
        }
    }

    createUnitTestDashboard(metrics: Array<Metric>): void {
        let cc = new GraphConfiguration(GraphType.AREA_STACKED);
        cc.title = this._translate.instant('graph_unittest_title');
        cc.colorScheme = { domain: [] };
        cc.gradient = false;
        cc.showXAxis = true;
        cc.showYAxis = true;
        cc.showLegend = true;
        cc.showXAxisLabel = true;
        cc.showYAxisLabel = true;
        cc.xAxisLabel = this._translate.instant('graph_unittest_x');
        cc.yAxisLabel = this._translate.instant('graph_unittest_y');
        cc.datas = new Array<ChartData>();

        let lines = ['ok', 'ko', 'skip'];
        lines.forEach(l => {
            let cd = new ChartData();
            cd.name = l;
            cd.series = new Array<ChartSeries>();
            metrics.forEach(m => {
                let v = m.value[l];
                if (v) {
                    let cs = new ChartSeries();
                    cs.name = m.run.toString();
                    cs.value = v;
                    cd.series.push(cs);
                }
            });
            cc.datas.push(cd);
            cc.colorScheme['domain'].push(Tests.getColor(l));
        });
        this.dashboards.push(cc);
    }

    createVulnDashboard(metrics: Array<Metric>): void {
        let cc = new GraphConfiguration(GraphType.AREA_STACKED);
        cc.title = this._translate.instant('graph_vulnerability_title');
        cc.colorScheme = { domain: [] };
        cc.gradient = false;
        cc.showXAxis = true;
        cc.showYAxis = true;
        cc.showLegend = true;
        cc.showXAxisLabel = true;
        cc.showYAxisLabel = true;
        cc.xAxisLabel = this._translate.instant('graph_vulnerability_x');
        cc.yAxisLabel = this._translate.instant('graph_vulnerability_y');
        cc.datas = new Array<ChartData>();

        Severity.Severities.forEach(s => {
            // Search for severity in datas
            let found = metrics.some(m => {
                if (m.value[s]) {
                    return true;
                }
            });
            if (found) {
                let cd = new ChartData();
                cd.name = s;
                cd.series = new Array<ChartSeries>();
                metrics.forEach(m => {
                    if (!m.run) {
                        return;
                    }
                    let v = m.value[s];
                    if (v) {
                        let cs = new ChartSeries();
                        cs.name = m.run.toString();
                        cs.value = v;
                        cd.series.push(cs);
                    }
                });
                cc.datas.push(cd);
                cc.colorScheme['domain'].push(Severity.getColors(s));
            }
        });
        this.dashboards.push(cc);
    }

    createCoverageDashboard(metrics: Array<Metric>): void {
        let cc = new GraphConfiguration(GraphType.AREA_STACKED);
        cc.title = this._translate.instant('graph_coverage_title');
        cc.colorScheme = { domain: [] };
        cc.gradient = false;
        cc.showXAxis = true;
        cc.showYAxis = true;
        cc.showLegend = false;
        cc.showXAxisLabel = true;
        cc.showYAxisLabel = true;
        cc.xAxisLabel = this._translate.instant('graph_vulnerability_x');
        cc.yAxisLabel = this._translate.instant('graph_coverage_y');
        cc.datas = new Array<ChartData>();

        let cd = new ChartData();
        cd.name = this._translate.instant('graph_coverage_y');
        cd.series = new Array<ChartSeries>();
        metrics.forEach(m => {
            let v = m.value['percent'];
            if (v) {
                let cs = new ChartSeries();
                cs.name = m.run.toString();
                cs.value = v;
                cd.series.push(cs);
            }
        });
        cc.datas.push(cd);
        cc.colorScheme['domain'].push('#4286f4');
        this.dashboards.push(cc);
    }
}
