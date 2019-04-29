use chrono::prelude::*;
use serde_json;
use serde::{Deserialize,Serialize};

use crate::models::{Group};
use crate::client::Client;
use crate::error::{Error as CdsError};

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Service {
    pub id: i64,
    pub name: String,
    pub r#type: String,
    pub http_url: String,
    pub last_heartbeat: Option<DateTime<Utc>>,
    pub hash: String,
    pub token: String,
    pub group_id: i64,
    pub group: Group,
    // pub monitoring_status: monitoring_status,
    pub config: serde_json::Value,
    pub is_shared_infra: bool,
    pub version: String,
    pub up_to_date: bool,
}

#[derive(Default, Debug)]
pub struct ServiceSDK {
    pub client: Client,
    pub hash: String,
    pub startup_time: Option<DateTime<Utc>>,
    pub api: String,
    pub name: String,
    pub http_url: String,
    pub token: String,
    pub r#type: String,
    pub max_heartbeat_failures: i32,
    pub service_name: String,
}

impl ServiceSDK {
    fn new() -> Self {
        ServiceSDK{
            ..Default::default()
        }
    }
}

pub trait ServiceTrait<T> {
    fn apply_configuration(&mut self, config: T) -> Result<(), CdsError>;
    fn check_configuration(&self, config: T) -> Result<(), CdsError>;
    fn heartbeat(&self, status: String, config: T) -> Result<(), CdsError>;
    fn register(&mut self, status: String, config: T) -> Result<(), CdsError>;
    fn status(&self) -> Result<String, CdsError>;
}

mod test {
    use super::*;
    use std::env;

    #[derive(Serialize, Deserialize, Default, Debug, Clone)]
    #[serde(default)]
    struct Configuration {
        pub name: String,
        pub url: String,
        pub api_url: String,
        pub token: String,
    }

    impl ServiceTrait<Configuration> for ServiceSDK {
        fn apply_configuration(&mut self, config: Configuration) -> Result<(), CdsError> {
            // TODO: convert client with String
            self.client = Client::new(config.api_url.clone(), "".to_string(), config.token.clone());
	        self.service_name = String::from("cds-service");
            self.api = config.api_url;
            self.name = config.name;
            self.token = config.token;
            Ok(())
        }

        fn check_configuration(&self, config: Configuration) -> Result<(), CdsError> {
            if config.token == "" {
                return Err(CdsError::from("token must not be empty".to_string()));
            }
            if config.api_url == "" {
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

        fn heartbeat(&self, status: String, config: Configuration) -> Result<(), CdsError> {
            Ok(())
        }

        fn register(&mut self, status: String, config: Configuration) -> Result<(), CdsError> {
            let mut srv = Service{
                name: self.name.to_owned(),
                r#type: self.r#type.to_owned(),
                http_url: self.http_url.to_owned(),
                last_heartbeat: Some(Utc::now()),
                token: self.token.to_owned(),
                config: serde_json::to_value(config).unwrap(),
                version: "Snapshot".to_string(),
                ..Default::default()
            };
            srv.hash = self.client.service_register(&srv)?;
            
            Ok(())
        }

        fn status(&self) -> Result<String, CdsError> {
            Ok("ok".to_string())
        }
    }


    #[test]
    fn test_service_create() {
        let my_conf = Configuration{
            name: "my_test".to_string(),
            url: "http://localhost:8088".to_string(),
            api_url: "http://localhost:8081".to_string(),
            token: env::var("CDS_SERVICE_TOKEN").unwrap()
        };
        let mut my_service = ServiceSDK{
            name: "test".to_string(),
            ..Default::default()
        };

        my_service.apply_configuration(my_conf.clone());

        assert_eq!(my_service.api, "http://localhost:8081".to_string());
        assert_eq!(my_service.register("OK".to_string(), my_conf).is_ok(), true);
    }
}
