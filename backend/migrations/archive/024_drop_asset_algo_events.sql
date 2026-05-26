-- Drop legacy asset_algo_events table. Replaced by asset_events (event_type IN ('algo_started','algo_finished','algo_failed')).
DROP TABLE IF EXISTS asset_algo_events;
