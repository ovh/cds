export class MonitoringVersion {
    version: string;
}

// MonitoringStatus contains status of CDS Component
export class MonitoringStatus {
    now: string;
    lines: Array<MonitoringStatusLine>;
}

// MonitoringStatusLine represents a CDS Component Status
export class MonitoringStatusLine {
    status: string;
    component: string;
    value: string;
    type: string;
    service: string;
    hostname: string;
    session: string;
    consumer: string;
}

export interface MonitoringMetricsLabel {
    name: string;
    value: string;
}

export interface MonitoringMetricsGauge {
    value: number;
}

export interface MonitoringMetricsMetric {
    label: MonitoringMetricsLabel[];
    gauge: MonitoringMetricsGauge;
}

export interface MonitoringMetricsLine {
    name: string;
    help: string;
    type: number;
    metric: MonitoringMetricsMetric[];
}
