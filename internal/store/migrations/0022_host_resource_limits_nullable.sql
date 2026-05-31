-- 将 hosts 表的资源限制列改为 nullable。
-- NULL = 无限制（Docker 层面不传资源参数）
-- 正值 = 具体限制
-- 现有数据保留不变（已有主机的 4096MB / 2.0 CPU / 20GB 是显式限制）
-- 默认值逻辑移至 API 层（见 Phase 57 D-01 / D-02）
ALTER TABLE hosts ALTER COLUMN memory_limit_mb DROP NOT NULL;
ALTER TABLE hosts ALTER COLUMN cpu_limit DROP NOT NULL;
ALTER TABLE hosts ALTER COLUMN disk_limit_gb DROP NOT NULL;
ALTER TABLE hosts ALTER COLUMN memory_limit_mb DROP DEFAULT;
ALTER TABLE hosts ALTER COLUMN cpu_limit DROP DEFAULT;
ALTER TABLE hosts ALTER COLUMN disk_limit_gb DROP DEFAULT;
