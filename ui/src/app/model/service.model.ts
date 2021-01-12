import { Group } from 'app/model/group.model';
import { MonitoringStatus } from './monitoring.model';


export class Global {
  name: string;
  status: string;
  value: string;
  services: Array<Service>;
}

export class Service {
  id: string;
  name: string;
  type: string;
  http_url: string;
  last_heartbeat: string;
  monitoring_status: MonitoringStatus;
  group_id: number;
  config: string;
  status: string;
  version: string;
  up_to_date: boolean;
  group: Group;
}
