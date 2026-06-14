
SET search_path TO public;

SELECT create_hypertable('sensor_data', 'timestamp', if_not_exists => TRUE, migrate_data => TRUE, chunk_time_interval => INTERVAL '1 hour');
SELECT create_hypertable('environmental_data', 'timestamp', if_not_exists => TRUE, migrate_data => TRUE, chunk_time_interval => INTERVAL '1 hour');

ALTER TABLE sensor_data SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'sensor_id',
    timescaledb.compress_orderby = 'timestamp DESC'
);

ALTER TABLE environmental_data SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'bridge_id',
    timescaledb.compress_orderby = 'timestamp DESC'
);

SELECT add_compression_policy('sensor_data', INTERVAL '7 days');
SELECT add_compression_policy('environmental_data', INTERVAL '7 days');

SELECT add_retention_policy('sensor_data', INTERVAL '2 years');
SELECT add_retention_policy('environmental_data', INTERVAL '2 years');

CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_data_hourly
WITH (timescaledb.continuous) AS
SELECT
    sensor_id,
    time_bucket('1 hour', timestamp) AS bucket,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    COUNT(*) AS sample_count,
    AVG(quality_flag) AS avg_quality
FROM sensor_data
GROUP BY sensor_id, time_bucket('1 hour', timestamp)
WITH NO DATA;

SELECT add_continuous_aggregate_policy('sensor_data_hourly',
    start_offset => INTERVAL '2 hours',
    end_offset   => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour'
);

CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_data_daily
WITH (timescaledb.continuous) AS
SELECT
    sensor_id,
    time_bucket('1 day', timestamp) AS bucket,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    COUNT(*) AS sample_count
FROM sensor_data
GROUP BY sensor_id, time_bucket('1 day', timestamp)
WITH NO DATA;

SELECT add_continuous_aggregate_policy('sensor_data_daily',
    start_offset => INTERVAL '2 days',
    end_offset   => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day'
);

CREATE INDEX IF NOT EXISTS idx_sensor_data_sensor_time ON sensor_data(sensor_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_bridge_time ON alerts(bridge_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_member_forces_analysis ON member_forces(analysis_id);
CREATE INDEX IF NOT EXISTS idx_node_displacements_analysis ON node_displacements(analysis_id);
CREATE INDEX IF NOT EXISTS idx_analysis_bridge_date ON analysis_results(bridge_id, analysis_date DESC);
