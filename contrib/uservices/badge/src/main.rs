// Http client
extern crate failure;
#[macro_use]
extern crate failure_derive;
extern crate serde_derive;
#[macro_use]
extern crate url_serde;
extern crate badge as badge_gen;
extern crate bytes;
extern crate chrono;
extern crate clap;
extern crate config;
extern crate core;
extern crate futures;
extern crate rdkafka;
extern crate rdkafka_sys;
extern crate serde;
extern crate serde_json;
extern crate url;

// Logger
#[macro_use]
extern crate log;
extern crate env_logger;

// Actors
extern crate actix;
extern crate actix_web;
extern crate tokio;

// Database
#[macro_use]
extern crate diesel;
extern crate diesel_migrations;
extern crate dotenv;
extern crate r2d2;
extern crate uuid;

// CDS
extern crate sdk_cds;

mod badge;
mod configuration;
mod database;
mod error;
mod kafka;
mod middlewares;
mod models;
mod run;
mod schema;
mod service;
mod web;
mod status;

use actix::prelude::*;
use actix::{Arbiter, SyncArbiter, System};
use clap::{App, Arg, SubCommand};
use diesel::prelude::PgConnection;
use diesel::r2d2::ConnectionManager;
use sdk_cds::service::ServiceTrait;

use database::DbExecutor;
use kafka::KafkaConsumerActor;
use web::WebState;

fn main() {
    env_logger::init();
    let config_subcmd = SubCommand::with_name("config")
        .about("Actions about configuration")
        .subcommand(
            SubCommand::with_name("new")
                .aliases(&["generate", "create"])
                .about("Generate your configuration file"),
        );
    let start_subcmd = SubCommand::with_name("start")
        .about("Start the µService")
        .arg(
            Arg::with_name("config")
                .short("f")
                .long("config")
                .value_name("FILE")
                .default_value("config.toml")
                .help("Sets a custom config file")
                .takes_value(true),
        );

    let app = App::new("CDS Badge Service")
        .version("1.0")
        .author("Benjamin Coenen <benjamin.coenen@corp.ovh.com>")
        .about("µService which generate badge for CDS workflow.")
        .subcommand(config_subcmd.clone())
        .subcommand(start_subcmd.clone());

    let matches = app.clone().get_matches();

    match matches.subcommand() {
        ("config", Some(config_cmd)) if config_cmd.subcommand_matches("new").is_some() => {
            println!("{}", configuration::get_example_config_file());
            return;
        }
        ("config", Some(_config_cmd)) => {
            config_subcmd.write_help(&mut std::io::stdout()).unwrap();
            return;
        }
        ("start", Some(start_cmd)) => {
            let config_arg = start_cmd.value_of("config").unwrap_or("config.toml");
            info!("Starting badge");
            let system = System::new("badge");
            let config = configuration::get_configuration(config_arg)
                .expect("Cannot set up the configuration");
            // Configure and start DB Executor actors
            let manager = ConnectionManager::<PgConnection>::new(format!(
                "postgres://{}:{}@{}:{}/{}",
                config.database.user,
                config.database.password,
                config.database.host,
                config.database.port,
                config.database.name
            ));
            let pool = r2d2::Pool::builder()
                .build(manager)
                .expect("Failed to create pool.");

            let addr = SyncArbiter::start(12, move || DbExecutor(pool.clone()));

            let brokers: Vec<String> = config.kafka.broker.split(',').map(String::from).collect();
            let db_addr = addr.clone();
            let kafka_config = config.kafka.clone();
            let _kafka_addr: Addr<KafkaConsumerActor> = Arbiter::start(|_| KafkaConsumerActor {
                brokers,
                topic: kafka_config.topic,
                group: kafka_config.group,
                user: kafka_config.user,
                password: kafka_config.password,
                db: db_addr,
            });

            let host = config.http.addr.clone();
            let port = config.http.port;
            let mut hash = String::new();
            let mode = config.mode.clone();

            if mode == "web" {
                let mut cds_service = service::new(config.clone());
                cds_service
                    .check_configuration(config.clone())
                    .expect("Cannot check configuration");
                cds_service
                    .apply_configuration(config.clone())
                    .expect("Cannot apply configuration");

                cds_service
                    .register(service::status(), config)
                    .expect("Cannot register service to CDS");

                hash = cds_service.service.hash.clone();
                let _ = Arbiter::start(|_| cds_service);
            }
            web::http_server(
                WebState {
                    db: addr.clone(),
                    hash,
                },
                host.clone(),
                port.to_string(),
            );

            println!("Server is listening on {}:{} in mode {}", host, port, mode);
            system.run();
        }
        _ => app.write_help(&mut std::io::stdout()).unwrap(),
    }
}
