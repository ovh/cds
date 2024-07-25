export class NodeStatus {
    static BUILDING = 'Building';
    static FAIL = 'Fail';
    static SUCCESS = 'Success';
    static WAITING = 'Waiting';
    static SCHEDULING = 'Scheduling';
    static SKIPPED = 'Skipped';
    static STOPPED = 'Stopped';

    static priority = [
        NodeStatus.SCHEDULING, NodeStatus.WAITING,
        NodeStatus.BUILDING, NodeStatus.STOPPED,
        NodeStatus.FAIL, NodeStatus.SUCCESS, NodeStatus.SKIPPED
    ];

    static neverRun(status: string) {
        return status === this.SKIPPED;
    }

    static isActive(status: string) {
        return status === this.WAITING || status === this.BUILDING || status === this.SCHEDULING;
    }

    static isDone(status: string) {
        return status === this.SUCCESS || status === this.STOPPED || status === this.FAIL ||
            status === this.SKIPPED;
    }

    static sum(status: Array<string>): string {
        const sum = status.map(s => NodeStatus.priority.indexOf(s)).reduce((sum, num) => {
            if (num > -1 && num < sum) { return num; }
            return sum;
        });
        if (sum === -1) {
            return null;
        }
        return NodeStatus.priority[sum];
    }
}

export enum GraphNodeAction {
    Enter = 'enter',
    Out = 'out',
    Click = 'click',
    ClickGate = 'click-gate',
    ClickRestart = 'click-restart',
    ClickStop = 'click-stop',
}
