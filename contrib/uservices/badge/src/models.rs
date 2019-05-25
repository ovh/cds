use schema::run;

#[derive(Debug, Default, Queryable, Insertable, QueryableByName, Serialize, Deserialize)]
#[table_name = "run"]
#[serde(default)]
pub struct Run {
    pub id: i64,
    pub run_id: i64,
    pub num: i64,
    pub project_key: String,
    pub workflow_name: String,
    pub branch: Option<String>,
    pub status: String,
    // pub updated: String,
}

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct Event {
  pub timestamp: String,
  pub hostname: String,
  pub cdsname: String,
  pub type_event: String,
  pub attempt: i64,
  pub project_key: String,
  pub workflow_name: String,
  pub workflow_run_num: i64,
  pub status: String,
  pub tag: Option<Vec<Tag>>,
}

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct Tag {
  pub tag: String,
  pub value: String,
}
