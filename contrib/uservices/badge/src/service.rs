use actix::prelude::*;
use chrono::prelude::*;
use sdk_cds::client::Client;
use sdk_cds::error::Error as CdsError;
use sdk_cds::models::{MonitoringStatus, StatusLine};
use sdk_cds::service::{Service as ServiceModel, ServiceSDK, ServiceTrait};
use serde_json;

use crate::configuration::BadgeConfiguration;

pub struct Service {
    pub service: ServiceSDK,
    pub config: BadgeConfiguration,
    current_heartbeat_failures: i32,
}

impl Default for Service {
    fn default() -> Service {
        Service {
            service: ServiceSDK::new(),
            config: BadgeConfiguration::default(),
            current_heartbeat_failures: 0,
        }
    }
}

impl Actor for Service {
    type Context = Context<Self>;

    fn started(&mut self, ctx: &mut Self::Context) {
        ctx.run_interval(
            std::time::Duration::new(30, 0),
            |service: &mut Service, _ctx: &mut Context<Self>| {
                if let Err(error) = service.heartbeat(status()) {
                    eprintln!("Error heartbeat {}", error);
                }
            },
        );
    }

    fn stopping(&mut self, _ctx: &mut Self::Context) -> Running {
        Running::Stop
    }
}

impl ServiceTrait<BadgeConfiguration> for Service {
    fn apply_configuration(&mut self, config: BadgeConfiguration) -> Result<(), CdsError> {
        self.service.client = Client::new(
            config.api.http.url.clone(),
            "cds-service-badge".to_string(),
            config.api.token.clone(),
        );
        self.service.service_name = String::from("cds-service-badge");
        self.service.api = config.api.http.url;
        self.service.name = config.name;
        self.service.token = config.api.token;
        self.service.r#type = String::from("badge");
        Ok(())
    }

    fn check_configuration(&self, config: BadgeConfiguration) -> Result<(), CdsError> {
        if config.api.token == "" {
            return Err(CdsError::from("token must not be empty".to_string()));
        }
        if config.api.http.url == "" {
            return Err(CdsError::from("api_url must not be empty".to_string()));
        }
        if config.url == "" {
            return Err(CdsError::from("url must not be empty".to_string()));
        }
        if config.name == "" {
            return Err(CdsError::from("name must not be empty".to_string()));
        }
        Ok(())
    }

    fn heartbeat(&mut self, status: MonitoringStatus) -> Result<(), CdsError> {
        if self.current_heartbeat_failures > self.service.max_heartbeat_failures {
            return Err(CdsError::from(String::from("max heartbeat failures")));
        }
        let conf = self.config.clone();
        if let Err(error) = self.register(status, conf) {
            self.current_heartbeat_failures += 1;
            eprintln!("Cannot heartbeat : {}", error);
        }
        Ok(())
    }

    fn register(
        &mut self,
        status: MonitoringStatus,
        config: BadgeConfiguration,
    ) -> Result<(), CdsError> {
        let srv = ServiceModel {
            name: self.service.name.to_owned(),
            r#type: self.service.r#type.to_owned(),
            http_url: self.service.http_url.to_owned(),
            last_heartbeat: Some(Utc::now()),
            token: self.service.token.to_owned(),
            config: serde_json::to_value(config).unwrap(),
            monitoring_status: status,
            version: "Snapshot".to_string(),
            ..Default::default()
        };
        self.service.hash = self.service.client.service_register(&srv)?;
        self.service.startup_time = Some(Utc::now());

        Ok(())
    }

    fn status(&self) -> MonitoringStatus {
        status()
    }
}

pub fn status() -> MonitoringStatus {
    let lines = vec![
        StatusLine {
            status: String::from("OK"),
            value: String::from("snapshot"),
            component: String::from("Version"),
            _type: String::new(),
        },
        StatusLine {
            status: String::from("OK"),
            value: String::from("Time"),
            component: format!("{}", Utc::now()),
            _type: String::new(),
        },
    ];
    MonitoringStatus {
        now: Some(Utc::now()),
        lines: Some(lines),
    }
}

pub fn new(config: BadgeConfiguration) -> Service {
    Service {
        service: ServiceSDK::new(),
        config,
        current_heartbeat_failures: 0,
    }
}
