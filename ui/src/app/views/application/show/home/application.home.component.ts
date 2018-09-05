import {Component, Input, OnInit} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Application, Severity} from '../../../../model/application.model';
import {ChartData, ChartSeries, GraphConfiguration, GraphType} from '../../../../model/graph.model';
import {Metric} from '../../../../model/metric.model';
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

    constructor(private _appNoCache: ApplicationNoCacheService, private _translate: TranslateService) {

    }

    ngOnInit(): void {
        this.dashboards = new Array<GraphConfiguration>();
        this._appNoCache.getMetrics(this.project.key, this.application.name, 'Vulnerability').subscribe(d => {
            if (d && d.length > 0) {
                this.createVulnDashboard(d);
            }
            this.ready = true;
        });
    }

    createVulnDashboard(metrics: Array<Metric>): void {
        let cc = new GraphConfiguration(GraphType.AREA_STACKED);
        cc.title = this._translate.instant('graph_vulnerability_title');
        cc.colorScheme = { domain: []};
        cc.gradient = false;
        cc.showXAxis = true;
        cc.showYAxis = true;
        cc.showLegend = true;
        cc.showXAxisLabel = false;
        cc.showYAxisLabel = false;
        cc.xAxisLabel = '';
        cc.yAxisLabel = '';
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
                    let v = m.value[s];
                    if (v) {
                        let cs = new ChartSeries();
                        cs.name = m.timestamp;
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
