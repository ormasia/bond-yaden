-- 债券行情明细表（记录所有原始行情数据）
CREATE TABLE t_bond_quote_detail (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '主键ID',
  message_id VARCHAR(64) NOT NULL COMMENT '消息ID',
  message_type VARCHAR(32) NOT NULL COMMENT '消息类型',
  timestamp BIGINT NOT NULL COMMENT '时间戳',
  security_code VARCHAR(32) NOT NULL COMMENT '债券代码',
  broker_id VARCHAR(32) NOT NULL COMMENT '券商ID',
  side VARCHAR(8) NOT NULL COMMENT '方向(BID/ASK)',
  price DECIMAL(18,6) NOT NULL COMMENT '报价',
  yield DECIMAL(18,6) COMMENT '收益率',
  order_qty DECIMAL(18,2) NOT NULL COMMENT '数量',
  min_trans_quantity DECIMAL(18,2) COMMENT '最小交易量',
  quote_order_no VARCHAR(64) NOT NULL COMMENT '报价单号',
  quote_time DATETIME NOT NULL COMMENT '报价时间',
  settle_type VARCHAR(16) COMMENT '结算类型',
  settle_date DATE COMMENT '结算日期',
  is_valid CHAR(1) COMMENT '是否有效(Y/N)',
  is_tbd CHAR(1) COMMENT '是否待定(Y/N)',
  create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_security_code (security_code),
  INDEX idx_quote_time (quote_time),
  INDEX idx_message_id (message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='债券行情明细表';

-- 债券最新行情表（仅记录每支债券最新行情）
CREATE TABLE t_bond_latest_quote (
  security_code VARCHAR(32) PRIMARY KEY COMMENT '债券代码',
  last_update_time DATETIME NOT NULL COMMENT '最后更新时间',
  bid_price DECIMAL(18,6) COMMENT '最优买入价',
  bid_yield DECIMAL(18,6) COMMENT '买入收益率',
  bid_qty DECIMAL(18,2) COMMENT '买入数量',
  bid_broker_id VARCHAR(32) COMMENT '买入券商ID',
  bid_quote_time DATETIME COMMENT '买入报价时间',
  ask_price DECIMAL(18,6) COMMENT '最优卖出价',
  ask_yield DECIMAL(18,6) COMMENT '卖出收益率',
  ask_qty DECIMAL(18,2) COMMENT '卖出数量',
  ask_broker_id VARCHAR(32) COMMENT '卖出券商ID',
  ask_quote_time DATETIME COMMENT '卖出报价时间',
  spread DECIMAL(18,6) GENERATED ALWAYS AS (ask_price - bid_price) STORED COMMENT '买卖价差',
  update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='债券最新行情表';