package db

const schemaSQL = `
CREATE TABLE IF NOT EXISTS products (
  id TEXT PRIMARY KEY,
  name TEXT,
  is_main INTEGER,
  issue_date TEXT,
  complete_date TEXT,
  subscribe_amount REAL,
  outstanding_amount REAL,
  manager TEXT,
  holding_status TEXT,
  structure_type TEXT,
  code TEXT,
  lock_days INTEGER,
  lock_months INTEGER,
  first_knockout_ratio REAL,
  entry_price REAL,
  monthly_decrease REAL,
  term TEXT,
  parachute TEXT,
  dividend_barrier REAL,
  monthly_coupon REAL,
  coupon_1st REAL,
  coupon_2nd REAL,
  coupon_3rd REAL,
  duration_months REAL,
  absolute_return REAL,
  holiday_adjust TEXT,
  raw TEXT,
  knock_in TEXT,
  duration_days INTEGER,
  knocked_in TEXT,
  margin_ratio REAL,
  custodian TEXT,
  counterparty TEXT
);

CREATE TABLE IF NOT EXISTS observations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT,
  observation_date TEXT,
  knockout_price REAL,
  dividend_line REAL,
  underlying_price REAL,
  is_knocked_out TEXT,
  is_dividend TEXT,
  months_since_entry INTEGER,
  updated_at TEXT,
  UNIQUE(product_id, observation_date),
  FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE IF NOT EXISTS price_cache (
  code TEXT,
  price_date TEXT,
  price REAL,
  updated_at TEXT,
  PRIMARY KEY (code, price_date)
);

CREATE TABLE IF NOT EXISTS sync_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  synced_at TEXT,
  row_count INTEGER
);

CREATE TABLE IF NOT EXISTS co_invest_users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_name TEXT,
  actual_buyer TEXT,
  phone TEXT,
  wechat TEXT,
  total_assets REAL,
  risk_tolerance TEXT,
  industry TEXT,
  is_actual_deal TEXT,
  lead_source TEXT,
  asset_match TEXT,
  is_dedicated_account TEXT,
  is_competitor TEXT,
  raw TEXT
);

CREATE TABLE IF NOT EXISTS co_invest_sync_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  synced_at TEXT,
  row_count INTEGER
);

CREATE TABLE IF NOT EXISTS customer_product_link (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT,
  user_name TEXT,
  actual_buyer TEXT
);

CREATE TABLE IF NOT EXISTS transactions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  transaction_date TEXT,
  flight_id TEXT,
  counterparty TEXT,
  subscribe_amount REAL,
  product_name TEXT,
  customer_name TEXT,
  actual_buyer TEXT,
  amount REAL,
  subscribe_fee_ratio REAL,
  management_fee_ratio REAL,
  performance_fee_ratio REAL,
  tax_subscribe_ratio REAL,
  tax_management_ratio REAL,
  tax_performance_ratio REAL,
  rebate_target TEXT,
  flight_date TEXT,
  holding_status TEXT,
  complete_date TEXT,
  underlying TEXT,
  structure_type TEXT,
  lock_period TEXT,
  dividend_barrier REAL,
  monthly_coupon REAL,
  coupon_1st REAL,
  raw TEXT,
  order_id TEXT
);

CREATE TABLE IF NOT EXISTS rebate_status (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id TEXT NOT NULL UNIQUE,
  is_returnable TEXT DEFAULT '',
  plan_subscribe INTEGER DEFAULT 0,
  plan_management INTEGER DEFAULT 0,
  plan_performance INTEGER DEFAULT 0,
  updated_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS rebate_completed (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id TEXT,
  flight_id TEXT,
  product_name TEXT,
  customer_name TEXT,
  channel_or_direct TEXT,
  principal REAL,
  margin_ratio REAL,
  business_type TEXT,
  subscribe_date TEXT,
  order_status TEXT,
  rebate_target TEXT,
  channel_subscribe_ratio REAL,
  channel_management_ratio REAL,
  channel_performance_ratio REAL,
  expense_category TEXT,
  expense_amount REAL,
  payment_time TEXT,
  payment_year TEXT,
  payment_month TEXT,
  payment_day TEXT,
  source TEXT DEFAULT 'manual',
  created_at TEXT DEFAULT (datetime('now')),
  updated_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS channels (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  channel_name TEXT,
  raw TEXT
);

CREATE TABLE IF NOT EXISTS direct_customer_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_name TEXT,
  raw TEXT
);

CREATE TABLE IF NOT EXISTS customers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  customer_name TEXT,
  raw TEXT
);

CREATE TABLE IF NOT EXISTS product_docs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_token TEXT UNIQUE,
  doc_name TEXT,
  parent_path TEXT,
  folder_token TEXT,
  raw_content TEXT,
  structure_json TEXT,
  synced_at TEXT
);

CREATE TABLE IF NOT EXISTS product_docs_sync_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  synced_at TEXT,
  doc_count INTEGER,
  folder_count INTEGER
);

CREATE TABLE IF NOT EXISTS posters (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id TEXT,
  poster_type TEXT,
  observation_date TEXT,
  product_name TEXT,
  date_display TEXT,
  months_since_entry INTEGER,
  underlying_name TEXT,
  absolute_return REAL,
  annualized_return REAL,
  duration_months INTEGER,
  parachute_value TEXT,
  knockout_value TEXT,
  dividend_barrier_value TEXT,
  dividend_count INTEGER,
  cumulative_rate REAL,
  monthly_coupon REAL,
  entry_date TEXT,
  created_at TEXT,
  UNIQUE(product_id, poster_type, observation_date),
  FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE IF NOT EXISTS activity_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  action TEXT NOT NULL,
  detail TEXT,
  created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS push_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  webhook_url TEXT NOT NULL DEFAULT '',
  cron_hour INTEGER NOT NULL DEFAULT 9,
  cron_minute INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 0,
  last_push_time TEXT DEFAULT NULL,
  last_push_result TEXT DEFAULT NULL,
  updated_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS agent_conversations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL DEFAULT '新对话',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS agent_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  conversation_id INTEGER NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  tool_calls TEXT DEFAULT NULL,
  tool_call_id TEXT DEFAULT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (conversation_id) REFERENCES agent_conversations(id) ON DELETE CASCADE
);
`
