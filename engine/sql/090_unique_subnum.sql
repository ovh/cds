-- +migrate Up
select create_unique_index('workflow_node_run', 'IDX_WORKFLOW_NODE_RUN_SUBNUM', 'workflow_node_id,num,sub_num');

DELETE FROM workflow_node_run WHERE id IN (SELECT id FROM (

select id from (
  select scount.*, wnr.id  from (
    select count(*) as nb, workflow_node_id,num,sub_num from workflow_node_run group by workflow_node_id,num,sub_num
  ) scount, workflow_node_run wnr
  where scount.nb > 1
  and wnr.workflow_node_id = scount.workflow_node_id and wnr.num = scount.num and wnr.sub_num = scount.sub_num
  ) allid

EXCEPT

SELECT MAX(id) from (
  select scount.*, wnr.id  from (
    select count(*) as nb, workflow_node_id,num,sub_num from workflow_node_run group by workflow_node_id,num,sub_num
  ) scount, workflow_node_run wnr
  where scount.nb > 1
  and wnr.workflow_node_id = scount.workflow_node_id and wnr.num = scount.num and wnr.sub_num = scount.sub_num
) maxid GROUP BY (maxid.workflow_node_id, maxid.num, maxid.sub_num)

) a );

-- +migrate Down
DROP INDEX IDX_WORKFLOW_NODE_RUN_SUBNUM;