-- Remove upload-related fields from analyses table

ALTER TABLE analyses
DROP INDEX idx_source_type,
DROP COLUMN start_file,
DROP COLUMN upload_id,
DROP COLUMN source_type;
