
use config::{Config, ConfigError, Environment, File};
use std::str::FromStr;
#[derive(Default, Debug, Deserialize, Clone)]
#[serde(default)]
pub struct Configuration {
  pub badge: BadgeConfiguration,
}

#[derive(Debug, Deserialize, Serialize, Clone)]
#[serde(default)]
pub struct BadgeConfiguration {
  #[serde(with = "url_serde")]
  pub url: url::Url,
  pub name: String,
  pub database: DatabaseConfiguration,
  pub kafka: KafkaConfiguration,
  pub http: HTTPConfiguration,
}

impl std::default::Default for BadgeConfiguration {
  fn default() -> Self {
    BadgeConfiguration {
      url: url::Url::from_str("http://localhost:8086").unwrap(),
      name: String::default(),
      database: DatabaseConfiguration::default(),
      kafka: KafkaConfiguration::default(),
      http: HTTPConfiguration::default(),
    }
  }
}

#[derive(Default, Debug, Deserialize, Serialize, Clone)]
#[serde(default)]
pub struct DatabaseConfiguration {
  pub user: String,
  pub password: String,
  pub name: String,
  pub host: String,
  pub port: i32,
  pub sslmode: String,
  pub maxconn: i32,
  pub timeout: i32,
}

#[derive(Default, Debug, Deserialize, Serialize, Clone)]
#[serde(default)]
pub struct KafkaConfiguration {
  pub group: String,
  pub user: String,
  pub password: String,
  pub broker: String,
  pub topic: String,
}

#[derive(Default, Debug, Deserialize, Serialize, Clone)]
pub struct HTTPConfiguration {
  #[serde(default = "default_addr")]
  pub addr: String,
  #[serde(default)]
  pub port: i32,
}

fn default_addr() -> String {
  "0.0.0.0".to_string()
}

pub fn get_configuration(filename: &str) -> Result<BadgeConfiguration, ConfigError> {
  let mut settings = Config::default();
  settings
    .merge(File::with_name(filename))?
    .merge(Environment::with_prefix("CDS").separator("_"))?;

  let conf: Configuration = settings.try_into()?;
  Ok(conf.badge)
}

pub fn get_example_config_file() -> &'static str {
  r#"#############################
# CDS Badge Service Settings
#############################
[badge]
  url = "http://localhost:8086"

  # Name of this CDS badge Service
  name = "cds-badge"

  ######################
  # CDS Badge Database Settings (postgresql)
  #######################
  [badge.database]
    user = ""
    password = ""
    name = ""
    host = "localhost"
    port = 5432
    maxconn = 20
    timeout = 3000

  ######################
  # CDS Badge Kafka Settings
  #######################
  [badge.kafka]
    broker = "localhost:9092"
    password = ""
    topic = "cds"
    user = ""
    group = "" # optional

  ######################
  # CDS Badge HTTP Configuration
  #######################
  [badge.http]

    # Listen address without port, example: 127.0.0.1
    # addr = ""
    port = 8088"#
}
