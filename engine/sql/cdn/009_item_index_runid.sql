-- +migrate Up

CREATE INDEX idx_item_runid ON item ((api_ref->>'run_id'));

-- +migrate Down
DROP index idx_item_runid;

