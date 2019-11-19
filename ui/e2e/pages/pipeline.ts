import config from '../config';
import { ClientFunction, t } from 'testcafe';

export default class PipelinePage {

    url: string;
    suffix: string

    getLocation = ClientFunction(() => document.location.href);

    constructor(key: string, pipName: string) {
        this.url = config.baseUrl + '/project/' + key + '/pipeline/' + pipName;
    }

    async go(suffix?: string) {
        await t
            .navigateTo(this.url + (suffix?suffix:''));
    }
}
