ALTER TABLE recurring_allowances DROP CONSTRAINT recurring_allowances_anchor_midnight;
ALTER TABLE allowance_revisions
    ADD COLUMN interval_unit text NOT NULL DEFAULT 'month',
    ADD COLUMN interval_count integer NOT NULL DEFAULT 1,
    ADD COLUMN schedule_anchor_at timestamptz;
UPDATE allowance_revisions r SET schedule_anchor_at=a.anchor_at
FROM recurring_allowances a WHERE a.id=r.allowance_id;
ALTER TABLE allowance_revisions
    ALTER COLUMN schedule_anchor_at SET NOT NULL,
    ADD CONSTRAINT allowance_interval_unit_valid CHECK (interval_unit IN ('minute','hour','day','week','month','year')),
    ADD CONSTRAINT allowance_interval_count_valid CHECK (interval_count BETWEEN 1 AND 1000000);
ALTER TABLE allowance_periods ALTER COLUMN skipped_periods TYPE bigint;
---- create above / drop below ----
-- Refuse a lossy downgrade rather than changing existing schedules or anchors.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM allowance_revisions WHERE interval_unit<>'month' OR interval_count<>1)
 OR EXISTS(SELECT 1 FROM recurring_allowances WHERE anchor_at<>date_trunc('day',anchor_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC')
 THEN RAISE EXCEPTION 'custom allowance schedules require migration 021'; END IF;
END $$;
ALTER TABLE allowance_periods ALTER COLUMN skipped_periods TYPE integer;
ALTER TABLE allowance_revisions DROP CONSTRAINT allowance_interval_unit_valid,
 DROP CONSTRAINT allowance_interval_count_valid,
 DROP COLUMN interval_unit,DROP COLUMN interval_count,DROP COLUMN schedule_anchor_at;
ALTER TABLE recurring_allowances ADD CONSTRAINT recurring_allowances_anchor_midnight
 CHECK (anchor_at=date_trunc('day',anchor_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC');
