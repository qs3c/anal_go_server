-- Add upload-related fields to analyses table for file upload analysis support

ALTER TABLE analyses
ADD COLUMN source_type VARCHAR(20) DEFAULT 'github' COMMENT '数据来源: github 或 upload' AFTER model_name,
ADD COLUMN upload_id VARCHAR(64) COMMENT '上传文件ID' AFTER source_type,
ADD COLUMN start_file VARCHAR(500) COMMENT '起始文件路径' AFTER upload_id;

-- Update existing records to have explicit source_type
UPDATE analyses SET source_type = 'github' WHERE source_type IS NULL;

-- Add index for source_type queries
ALTER TABLE analyses ADD INDEX idx_source_type (source_type);
