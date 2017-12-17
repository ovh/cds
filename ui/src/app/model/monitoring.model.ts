

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
}
