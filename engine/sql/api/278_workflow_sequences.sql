-- +migrate Up
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION v2_workflow_run_sequences_nextval(repositoryID text, workflowName text) RETURNS integer AS $$
DECLARE
  workflow_exists integer;
  cur_val integer;
  repositoryUUID uuid;
BEGIN
  SELECT CAST(repositoryID as uuid) INTO repositoryUUID;

  SELECT    COUNT(1) INTO workflow_exists
  FROM      v2_workflow_run_sequences
  WHERE     repository_id = repositoryUUID AND workflow_name = workflowName;

  IF workflow_exists = 0 THEN
    BEGIN
      INSERT INTO v2_workflow_run_sequences(repository_id, workflow_name, current_val) VALUES (repositoryUUID, workflowName, 0);
    EXCEPTION WHEN others THEN
    -- Do nothing
    END;
  END IF;

  SELECT    current_val INTO cur_val
  FROM      v2_workflow_run_sequences
  WHERE     repository_id = repositoryUUID AND workflow_name = workflowName FOR UPDATE;

  UPDATE    v2_workflow_run_sequences SET current_val = cur_val + 1 WHERE repository_id = repositoryUUID AND workflow_name = workflowName;
  RETURN    cur_val + 1;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate Down
