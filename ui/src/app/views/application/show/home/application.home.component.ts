import {Component, Input, OnInit} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Application, Overview, Severity} from '../../../../model/application.model';
import {ChartData, ChartSeries, GraphConfiguration, GraphType} from '../../../../model/graph.model';
import {Metric} from '../../../../model/metric.model';
import {Tests} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {ApplicationNoCacheService} from '../../../../service/application/application.nocache.service';

@Component({
    selector: 'app-home',
    templateUrl: './application.home.html',
    styleUrls: ['./application.home.scss']
})
export class ApplicationHomeComponent implements OnInit {

    @Input() project: Project;
    @Input() application: Application;

    dashboards: Array<GraphConfiguration>;
    ready = false;
    overview: Overview;

    constructor(private _appNoCache: ApplicationNoCacheService, private _translate: TranslateService) {

    }

    ngOnInit(): void {
        this.dashboards = new Array<GraphConfiguration>();
        this._appNoCache.getOverview(this.project.key, this.application.name).subscribe(d => {
            this.overview = d;
            if (d && d.graphs.length > 0) {
                d.graphs.forEach(g => {
                    switch (g.type) {
                        case 'Vulnerability':
                            this.createVulnDashboard(g.datas);
                            break;
                        case 'UnitTest':
                            this.createUnitTestDashboard(g.datas);
                            break;
                    }
                });
            }
            this.ready = true;
        });
    }

    createUnitTestDashboard(metrics: Array<Metric>): void {
        let cc = new GraphConfiguration(GraphType.AREA_STACKED);
        cc.title = this._translate.instant('graph_unittest_title');
        cc.colorScheme = { domain: []};
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
        cc.colorScheme = { domain: []};
        cc.gradient = false;
        cc.showXAxis = true;
        cc.showYAxis = true;
        cc.showLegend = true;
        cc.showXAxisLabel = true;
        cc.showYAxisLabel = true;
        cc.xAxisLabel = this._translate.instant('graph_vulnerability_x');
        cc.yAxisLabel = this._translate.instant('graph_vulnerability_y'); ;
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
}
