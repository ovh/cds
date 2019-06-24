use actix::{Actor, Addr, Context, Running};
use futures::{Future, Stream};
use rdkafka::client::ClientContext;
use rdkafka::config::{ClientConfig, RDKafkaLogLevel};
use rdkafka::consumer::stream_consumer::StreamConsumer;
use rdkafka::consumer::{CommitMode, Consumer, ConsumerContext, Rebalance};
use rdkafka::error::KafkaResult;
use rdkafka::message::Message;
use rdkafka_sys;
use serde_json;


use crate::database::DbExecutor;
use crate::models::{Event, Run};
use crate::run::CreateRun;
use crate::web::Pool;


#[derive(Clone)]
pub struct KafkaConsumerActor {
    pub brokers: Vec<String>,
    pub topic: String,
    pub user: String,
    pub password: String,
    pub group: String,
    pub db: Pool,
    pub db_actor: Addr<DbExecutor>,
}

struct CustomContext;
impl ClientContext for CustomContext {}
impl ConsumerContext for CustomContext {
    fn pre_rebalance(&self, rebalance: &Rebalance) {
        info!("Pre rebalance {:?}", rebalance);
    }

    fn post_rebalance(&self, rebalance: &Rebalance) {
        info!("Post rebalance {:?}", rebalance);
    }

    fn commit_callback(
        &self,
        result: KafkaResult<()>,
        _offsets: *mut rdkafka_sys::RDKafkaTopicPartitionList,
    ) {
        info!("Committing offsets: {:?}", result);
    }
}

impl Actor for KafkaConsumerActor {
    type Context = Context<Self>;

    fn started(&mut self, _ctx: &mut Self::Context) {
        println!("Kafka consumer starting");
        let consumer = create_consumer(
            self.user.clone(),
            self.password.clone(),
            self.group.clone(),
            self.brokers.clone(),
        )
        .expect("cannot create kafka consumer");

        let topics = [self.topic.as_str()];
        consumer
            .subscribe(&topics.to_vec())
            .expect("Can't subscribe to specified topics");

        println!(
            "Connected to the kafka {:?} on topic {}",
            self.brokers, self.topic
        );

        // consumer.start() returns a stream. The stream can be used ot chain together expensive steps,
        // such as complex computations on a thread pool or asynchronous IO.
        let message_stream = consumer.start();
        for message in message_stream.wait().flatten() {
            match message {
                Err(err) => eprintln!("Error while reading from stream. : {:?}", err),
                Ok(msg) => {
                    let payload = msg.payload();
                    if payload.is_none() {
                        continue;
                    }
                    match serde_json::from_slice::<Event>(payload.unwrap()) {
                        Ok(ref event) if event.type_event == "sdk.EventRunWorkflow" => {
                            let mut branch = None;
                            if let Some(ref tags) = &event.tag {
                                branch = tags
                                    .iter()
                                    .find(|tag| tag.tag == "git.branch")
                                    .map(|tag| tag.value.clone());
                            }

                            let run = Run {
                                num: event.workflow_run_num,
                                project_key: event.project_key.clone(),
                                workflow_name: event.workflow_name.clone(),
                                branch,
                                status: event.status.clone().into(),
                                ..Default::default()
                            };

                            if let Err(err) = self.db_actor.send(CreateRun { run }).flatten().wait()
                            {
                                eprintln!("future run NOT created in db {:?}", err);
                            }

                            debug!("{:#?}", event);
                        }
                        _ => (),
                    }

                    consumer
                        .commit_message(&msg, CommitMode::Async)
                        .expect("cannot commit message");
                }
            };
        }
    }

    fn stopping(&mut self, _ctx: &mut Self::Context) -> Running {
        Running::Stop
    }
}

fn create_consumer(
    user: String,
    password: String,
    group: String,
    brokers: Vec<String>,
) -> KafkaResult<StreamConsumer<CustomContext>> {
    let context = CustomContext;
    let mut client_config = ClientConfig::new();

    client_config
        .set_log_level(RDKafkaLogLevel::Debug)
        .set("metadata.broker.list", &brokers.join(","))
        .set("enable.partition.eof", "false")
        .set("api.version.request", "true")
        .set("debug", "protocol,security,broker")
        .set("session.timeout.ms", "6000")
        .set("auto.offset.reset", "latest")
        .set("enable.auto.commit", "true");

    if group != "" {
        client_config.set("group.id", &group);
    } else {
        client_config.set("group.id", &(user.clone() + ".cds-badge"));
    }

    if user != "" && password != "" {
        client_config
            .set("security.protocol", "SASL_SSL")
            .set("sasl.mechanisms", "PLAIN")
            .set("sasl.username", &user)
            .set("sasl.password", &password);
    }

    client_config.create_with_context(context)
}
