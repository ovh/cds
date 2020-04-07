-- +migrate Up

CREATE UNIQUE INDEX idx_w_node_hook_uuid 
ON w_node_hook (uuid);

ALTER TABLE w_node_hook 
ADD CONSTRAINT unique_w_node_hook_uuid  
UNIQUE USING INDEX idx_w_node_hook_uuid;

-- +migrate Down

SELECT 1;
