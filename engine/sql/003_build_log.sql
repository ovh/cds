-- +migrate Up
DROP TABLE pipeline_history_old;

CREATE TABLE IF NOT EXISTS "pipeline_build_log" (
  id BIGSERIAL PRIMARY KEY,
  pipeline_build_job_id BIGINT,
  pipeline_build_id BIGINT,
  start TIMESTAMP WITH TIME ZONE,
  last_modified TIMESTAMP WITH TIME ZONE,
  done TIMESTAMP WITH TIME ZONE,
  step_order BIGINT,
  "value" BYTEA
);
select create_foreign_key('FK_BUILD_LOG_PIPELINE_BUILD', 'pipeline_build_log', 'pipeline_build', 'pipeline_build_id', 'id');
select create_unique_index('pipeline_build_log', 'IDX_PIPELINE_BUILD_LOG_UNIQUE', 'pipeline_build_id,pipeline_build_job_id,step_order');

INSERT INTO pipeline_build_log (pipeline_build_job_id, pipeline_build_id, start, last_modified, done, step_order, value)
SELECT tmp.action_build_id, tmp.pipeline_build_id, max(b.timestamp), max(b.timestamp), max(b.timestamp), 0, convert_to(string_agg(b.value, ''), 'UTF8')
FROM (
	SELECT bl.id, bl.pipeline_build_id, bl.action_build_id
	FROM build_log bl
	LEFT JOIN pipeline_build_log pbl ON pbl.pipeline_build_id = bl.pipeline_build_id AND pbl.pipeline_build_job_id = bl.action_build_id
	WHERE bl.pipeline_build_id IS NOT NULL
	  AND bl.action_build_id IS NOT NULL
	  AND pbl.pipeline_build_job_id IS NULL
	ORDER by bl.id ASC
) tmp
JOIN build_log b ON b.id = tmp.id
GROUP BY tmp.pipeline_build_id, tmp.action_build_id
ORDER BY tmp.pipeline_build_id, tmp.action_build_id;

-- +migrate Down
DROP TABLE pipeline_build_log;


