import { Duration } from '@icholy/duration';

export class DurationService {
    public static duration(from: Date, to: Date): string {
        // zero date
        if (from.getFullYear() === 1) {
            return '0s';
        }
        let fromMs = Math.round(from.getTime() / 1000) * 1000;
        let toMs = Math.round(to.getTime() / 1000) * 1000;
        let sub = toMs - fromMs;
        return sub === 0 ? '~0s' : (new Duration(sub)).toString();
    }
}
