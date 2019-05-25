table! {
    run (id) {
        id -> Int8,
        run_id -> Int8,
        num -> Int8,
        project_key -> Text,
        workflow_name -> Text,
        branch -> Nullable<Text>,
        status -> Text,
        updated -> Nullable<Timestamptz>,
    }
}
