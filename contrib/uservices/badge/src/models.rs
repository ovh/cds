use schema::run;

#[derive(Debug, Serialize, Deserialize, PartialEq, Clone)]
pub enum StatusEnum {
  Pending,
  Waiting,
  Checking,
  Building,
  Success,
  Fail,
  Disabled,
  NeverBuilt,
  Unknown,
  Skipped,
  Stopped,
}

impl std::default::Default for StatusEnum {
  fn default() -> Self {
    StatusEnum::NeverBuilt
  }
}

impl Into<String> for StatusEnum {
  fn into(self) -> String {
    match self {
      StatusEnum::Pending => String::from("Pending"),
      StatusEnum::Waiting => String::from("Waiting"),
      StatusEnum::Checking => String::from("Checking"),
      StatusEnum::Building => String::from("Building"),
      StatusEnum::Success => String::from("Success"),
      StatusEnum::Fail => String::from("Fail"),
      StatusEnum::Stopped => String::from("Stopped"),
      StatusEnum::Disabled => String::from("Disabled"),
      StatusEnum::NeverBuilt => String::from("NeverBuilt"),
      StatusEnum::Unknown => String::from("Unknown"),
      StatusEnum::Skipped => String::from("Skipped"),
    }
  }
}

impl From<String> for StatusEnum {
  fn from(elt: String) -> Self {
    match elt.as_ref() {
      "Pending" => StatusEnum::Pending,
      "Waiting" => StatusEnum::Waiting,
      "Checking" => StatusEnum::Checking,
      "Building" => StatusEnum::Building,
      "Success" => StatusEnum::Success,
      "Fail" => StatusEnum::Fail,
      "Stopped" => StatusEnum::Stopped,
      "Disabled" => StatusEnum::Disabled,
      "NeverBuilt" => StatusEnum::NeverBuilt,
      "Unknown" => StatusEnum::Unknown,
      "Skipped" => StatusEnum::Skipped,
      _ => StatusEnum::Unknown,
    }
  }
}

impl From<&String> for StatusEnum {
  fn from(elt: &String) -> Self {
    match elt.as_ref() {
      "Pending" => StatusEnum::Pending,
      "Waiting" => StatusEnum::Waiting,
      "Checking" => StatusEnum::Checking,
      "Building" => StatusEnum::Building,
      "Success" => StatusEnum::Success,
      "Fail" => StatusEnum::Fail,
      "Stopped" => StatusEnum::Stopped,
      "Disabled" => StatusEnum::Disabled,
      "NeverBuilt" => StatusEnum::NeverBuilt,
      "Unknown" => StatusEnum::Unknown,
      "Skipped" => StatusEnum::Skipped,
      _ => StatusEnum::Unknown,
    }
  }
}

impl From<&str> for StatusEnum {
  fn from(elt: &str) -> Self {
    match elt {
      "Pending" => StatusEnum::Pending,
      "Waiting" => StatusEnum::Waiting,
      "Checking" => StatusEnum::Checking,
      "Building" => StatusEnum::Building,
      "Success" => StatusEnum::Success,
      "Fail" => StatusEnum::Fail,
      "Stopped" => StatusEnum::Stopped,
      "Disabled" => StatusEnum::Disabled,
      "NeverBuilt" => StatusEnum::NeverBuilt,
      "Unknown" => StatusEnum::Unknown,
      "Skipped" => StatusEnum::Skipped,
      _ => StatusEnum::Unknown,
    }
  }
}

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
  pub status: StatusEnum,
  pub tag: Option<Vec<Tag>>,
}

#[derive(Debug, Serialize, Deserialize, Default)]
pub struct Tag {
  pub tag: String,
  pub value: String,
}
