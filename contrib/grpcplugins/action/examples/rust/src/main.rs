use std::net::SocketAddr;

use tonic::{transport::Server, Request, Response, Status};

use action_plugin::action_plugin_server::{ActionPlugin, ActionPluginServer};
use action_plugin::{ActionPluginManifest, ActionQuery, ActionResult, WorkerHttpPortQuery};

pub mod action_plugin {
    tonic::include_proto!("actionplugin");
}

#[derive(Debug, Default)]
pub struct MyActionPlugin {
    worker_http_port: usize,
}

#[tonic::async_trait]
impl ActionPlugin for MyActionPlugin {
    async fn run(&self, request: Request<ActionQuery>) -> Result<Response<ActionResult>, Status> {
        let reply = ActionResult {
            status: String::from("Success"),
            details: String::new(),
        };

        println!(
            "{}",
            request
                .get_ref()
                .options
                .get("log")
                .unwrap_or(&String::from("Hello World!"))
        );

        Ok(Response::new(reply))
    }

    async fn worker_http_port(
        &self,
        _request: Request<WorkerHttpPortQuery>,
    ) -> Result<Response<()>, Status> {
        Ok(Response::new(()))
    }

    async fn manifest(
        &self,
        _request: Request<()>,
    ) -> Result<Response<ActionPluginManifest>, Status> {
        let reply = ActionPluginManifest {
            name: String::from("rust-plugin"),
            version: String::from("snapshot"),
            description: String::from("this is a test"),
            author: String::from("bnjjj"),
        };

        Ok(Response::new(reply))
    }

    async fn stop(&self, _request: Request<()>) -> Result<Response<()>, Status> {
        std::process::exit(0);
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr: SocketAddr = "127.0.0.1:50051".parse()?;
    let actionplugin = MyActionPlugin::default();

    println!("{} is ready to accept new connection", addr.to_string());
    Server::builder()
        .add_service(ActionPluginServer::new(actionplugin))
        .serve(addr)
        .await?;

    Ok(())
}
