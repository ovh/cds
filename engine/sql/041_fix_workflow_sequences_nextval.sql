-- +migrate Up
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION workflow_sequences_nextval(w_id integer) RETURNS integer AS $$
DECLARE
    workflow_exists integer;
    cur_val integer;
BEGIN
    SELECT    COUNT(1) INTO workflow_exists
    FROM      workflow_sequences
    WHERE     workflow_id = $1;

    IF workflow_exists = 0 THEN
        BEGIN
            INSERT INTO workflow_sequences(workflow_id, current_val) VALUES ($1, 0);
        EXCEPTION WHEN others THEN
            -- Do nothing
        END;
    END IF;
    
    SELECT    current_val INTO cur_val
    FROM      workflow_sequences
    WHERE     workflow_id = $1 FOR UPDATE;

    UPDATE    workflow_sequences SET current_val = cur_val + 1 WHERE workflow_id = $1;

    RETURN    cur_val + 1;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate Down
DROP FUNCTION workflow_sequences_nextval(integer);