CREATE TABLE subscriptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    plan ENUM('basic', 'pro') NOT NULL COMMENT '套餐',
    amount DECIMAL(10, 2) COMMENT '金额',
    daily_quota INT COMMENT '每日配额',
    started_at DATETIME NOT NULL COMMENT '生效时间',
    expires_at DATETIME NOT NULL COMMENT '过期时间',
    status ENUM('active', 'expired', 'cancelled') DEFAULT 'active',
    payment_method ENUM('wechat', 'alipay') COMMENT '支付方式',
    transaction_id VARCHAR(100) COMMENT '交易ID',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅记录表';
