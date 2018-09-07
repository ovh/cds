export class GraphConfiguration {
    type: string;
    title: string;
    colorScheme: {};
    gradient = false;
    showXAxis = true;
    showYAxis = true;
    showLegend = true;
    showXAxisLabel = true;
    showYAxisLabel = true;
    xAxisLabel = '';
    yAxisLabel = '';
    datas: Array<ChartData>;

    constructor(t: string) {
        this.type = t;
    }
}

export class GraphType {
    static AREA_STACKED = 'area-stacked';
}

export class ChartData {
    name: string;
    series: Array<ChartSeries>;
}

export class ChartSeries {
    name: string;
    value: number;
}
