-- Extend egress_ips with tunnel type differentiation and proxy configuration.
ALTER TABLE egress_ips
    ADD COLUMN tunnel_type TEXT NOT NULL DEFAULT 'wireguard';

ALTER TABLE egress_ips
    ADD CONSTRAINT egress_ips_tunnel_type_check
    CHECK (tunnel_type IN ('wireguard', 'proxy'));

ALTER TABLE egress_ips
    ADD COLUMN proxy_config JSONB;
