-- 更新债券最新行情表结构
-- 备份现有表（如果需要）
-- CREATE TABLE t_bond_latest_quote_backup AS SELECT * FROM t_bond_latest_quote;

-- 删除现有表并重新创建（注意：这会丢失现有数据）
DROP TABLE IF EXISTS t_bond_latest_quote;

-- 创建新的最新行情表（JSON存储）
CREATE TABLE t_bond_latest_quote (
    isin VARCHAR(32) PRIMARY KEY COMMENT '债券代码',
    raw_json TEXT NOT NULL COMMENT '完整消息JSON',
    message_id VARCHAR(64) NULL COMMENT '消息ID',
    message_type VARCHAR(32) NULL COMMENT '消息类型',
    send_time BIGINT NULL COMMENT '消息发送时间（毫秒）',
    timestamp BIGINT NULL COMMENT '业务时间戳',
    last_update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '最后更新时间',
    
    INDEX idx_message_id (message_id),
    INDEX idx_send_time (send_time),
    INDEX idx_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='债券最新行情表';

-- 更新明细表的字段名（如果需要）
-- ALTER TABLE t_bond_quote_detail CHANGE COLUMN isin isin VARCHAR(32) NOT NULL COMMENT '债券代码';

-- 添加索引优化查询性能
-- ALTER TABLE t_bond_quote_detail ADD INDEX idx_isin_quote_time (isin, quote_time);
-- ALTER TABLE t_bond_quote_detail ADD INDEX idx_broker_side (broker_id, side);
