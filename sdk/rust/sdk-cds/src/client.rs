use super::error::Error as CdsError;
use super::models;
use crate::service::Service;

use std::collections::HashMap;

use reqwest::Client as HttpClient;
use serde::de::DeserializeOwned;
use serde::Serialize;
use base64;

const SESSION_TOKEN_HEADER: &'static str = "Session-Token";

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub struct Client {
    pub name: String, //Useful for multi instance to give a name to your instance
    pub host: String,
    pub username: String,
    pub token: String,
    pub hash: String,
    pub insecure_skip_verify_tls: bool,
}

impl Client {
    pub fn new<T: Into<String>>(host: T, username: T, token: T) -> Self {
        let host: String = host.into();
        Client {
            host: host.clone(),
            username: username.into(),
            token: token.into(),
            insecure_skip_verify_tls: !host.starts_with("https"),
            ..Default::default()
        }
    }

    pub fn status(&self) -> Result<models::Status, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/mon/status".to_string(), body)
    }

    pub fn config(&self) -> Result<HashMap<String, String>, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/config/user".to_string(), body)
    }

    pub fn me(&self) -> Result<models::User, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), format!("/user/{}", self.username), body)
    }

    pub fn broadcasts(&self) -> Result<Vec<models::Broadcast>, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/broadcast".to_string(), body)
    }

    pub fn projects(&self) -> Result<Vec<models::Project>, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/project".to_string(), body)
    }

    pub fn applications(&self, project_key: &str) -> Result<Vec<models::Application>, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/applications", project_key),
            body,
        )
    }

    pub fn application(
        &self,
        project_key: &str,
        application_name: &str,
    ) -> Result<models::Application, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/application/{}", project_key, application_name),
            body,
        )
    }

    pub fn workflows(&self, project_key: &str) -> Result<Vec<models::Workflow>, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/workflows", project_key),
            body,
        )
    }

    pub fn workflow(
        &self,
        project_key: &str,
        workflow_name: &str,
    ) -> Result<models::Workflow, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/workflows/{}", project_key, workflow_name),
            body,
        )
    }

    pub fn queue_count(&self) -> Result<models::QueueCount, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            "/queue/workflows/count".to_string(),
            body,
        )
    }

    pub fn bookmarks(&self) -> Result<Vec<models::Bookmark>, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), String::from("/bookmarks"), body)
    }

    pub fn last_run(
        &self,
        project_key: &str,
        workflow_name: &str,
    ) -> Result<models::WorkflowRun, CdsError> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!(
                "/project/{}/workflows/{}/runs/latest",
                project_key, workflow_name
            ),
            body,
        )
    }

    pub fn service_register(&self, service: &Service) -> Result<String, CdsError> {
        self.stream_json(
            "POST".to_string(),
            "/services/register".to_string(),
            service
        ).map(|serv_resp: Service| serv_resp.hash)
    }

    pub fn stream_json<T: Serialize, U: DeserializeOwned>(
        &self,
        method: String,
        path: String,
        body: T,
    ) -> Result<U, CdsError> {
        let url = format!("{}{}", self.host, path);
        let mut req_http = HttpClient::new()
            .request(reqwest::Method::from_bytes(method.as_bytes())?, &url)
            .header(reqwest::header::CONTENT_TYPE, "application/json")
            .header(reqwest::header::USER_AGENT, "CDS/sdk")
            .header("X-Requested-With", "X-CDS-SDK")
            .header(SESSION_TOKEN_HEADER, self.token.clone());

        if self.username != "" {
            req_http = req_http.basic_auth(self.username.clone(), Some(self.token.clone()));
        }
        
        if self.hash != "" {
            req_http = req_http.header("X_AUTH_HEADER", base64::encode(&self.hash));
        }
        
        let mut resp_http = req_http.json(&body).send()?;

        if resp_http.status().as_u16() > 400u16 {
            let mut err: CdsError = resp_http.json().map_err(CdsError::from)?;
            err.status = resp_http.status().as_u16();
            return Err(err);
        }

        resp_http.json().map_err(CdsError::from)
    }
}
