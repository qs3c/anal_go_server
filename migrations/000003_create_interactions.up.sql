CREATE TABLE interactions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    analysis_id BIGINT NOT NULL COMMENT '分析ID',
    type ENUM('like', 'bookmark') NOT NULL COMMENT '互动类型',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE,
    UNIQUE KEY uk_user_analysis_type (user_id, analysis_id, type),
    INDEX idx_analysis_type (analysis_id, type),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='互动表';
