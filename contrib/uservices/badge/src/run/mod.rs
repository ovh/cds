pub mod handlers;

use actix::{Handler, Message};
use actix_web::{error, Error};
use diesel::insert_into;
use diesel::prelude::*;
use diesel::sql_query;
use diesel::sql_types::Text;

use super::database::DbExecutor;
use super::models::Run;

pub struct QueryRun {
    pub project_key: String,
    pub workflow_name: String,
    pub branch: Option<String>,
}

impl Message for QueryRun {
    type Result = Result<Run, Error>;
}

impl Handler<QueryRun> for DbExecutor {
    type Result = Result<Run, Error>;

    fn handle(&mut self, msg: QueryRun, _: &mut Self::Context) -> Self::Result {
        let conn: &PgConnection = &self.0.get().unwrap();
        let query = if msg.branch.is_none() {
            "SELECT * FROM run WHERE project_key = $1 AND workflow_name = $2 AND branch IS NULL ORDER BY (num, updated) DESC LIMIT 1"
        } else {
            "SELECT * FROM run WHERE project_key = $1 AND workflow_name = $2 AND branch = $3 ORDER BY (num, updated) DESC LIMIT 1"
        };

        let mut run_res = sql_query(query)
            .bind::<Text, _>(msg.project_key)
            .bind::<Text, _>(msg.workflow_name)
            .bind::<Text, _>(msg.branch.unwrap_or_default())
            .get_results(conn)
            .map_err(error::ErrorInternalServerError)?;

        if run_res.is_empty() {
            return Err(error::ErrorNotFound("not found"));
        }

        Ok(run_res.pop().unwrap())
    }
}

pub struct CreateRun {
    pub run: Run,
}

impl Message for CreateRun {
    type Result = Result<Run, Error>;
}

impl Handler<CreateRun> for DbExecutor {
    type Result = Result<Run, Error>;

    fn handle(&mut self, msg: CreateRun, _: &mut Self::Context) -> Self::Result {
        // use diesel::select;
        use schema::run::dsl::*;
        let conn: &PgConnection = &self.0.get().unwrap();
        // let run_inserted = insert_into(run)
        insert_into(run)
            .values((
                run_id.eq(&msg.run.run_id),
                num.eq(&msg.run.num),
                project_key.eq(&msg.run.project_key),
                workflow_name.eq(&msg.run.workflow_name),
                branch.eq(&msg.run.branch),
                status.eq(&msg.run.status),
            ))
            .execute(conn)
            // .get_result::<Run>(conn)
            .map_err(error::ErrorInternalServerError)?;

        Ok(msg.run)
    }
}
