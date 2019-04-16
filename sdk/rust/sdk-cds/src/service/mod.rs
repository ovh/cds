use chrono::prelude::*;
use serde_json;

use crate::models::{Group};
use crate::error::{CdsError};

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Service {
    pub id: i64,
    pub name: String,
    pub type: String,
    pub http_url: String,
    pub last_heartbeat: DateTime<Utc>,
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


ApplyConfiguration(cfg interface{}) error
	Serve(ctx context.Context) error
	CheckConfiguration(cfg interface{}) error
	Heartbeat(ctx context.Context, status func() sdk.MonitoringStatus, cfg interface{}) error
	Register(status func() sdk.MonitoringStatus, cfg interface{}) error
	Status() sdk.MonitoringStatus
pub trait Service {
    pub fn apply_configuration<T>(&self, config: T) -> Result<(), CdsError> where T: Deserialize+Serialize;
    pub fn serve(&self) -> Result<(), CdsError>;
    pub fn check_configuration<T>(&self, config: T) -> Result<(), CdsError> where T: Deserialize+Serialize;




    pub fn heartbeat<T>(&self, status: String, config: T) -> Result<(), CdsError> where T: Deserialize+Serialize {
        Ok(())
    }

    pub fn register<T>(&mut self, status: String, config: T) -> Result<(), CdsError> where T: Deserialize+Serialize {
        Ok(())
    }

    pub fn status(&self) -> Result<String, CdsError>;
}