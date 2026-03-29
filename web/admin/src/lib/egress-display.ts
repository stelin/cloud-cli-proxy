import type { EgressIP } from "@/hooks/use-egress-ips";

function proxyServerPort(config: Record<string, unknown>): number | undefined {
  const p = config.server_port;
  if (typeof p === "number" && Number.isFinite(p) && p > 0 && p <= 65535) {
    return p;
  }
  if (typeof p === "string") {
    const n = parseInt(p, 10);
    if (!Number.isNaN(n) && n > 0 && n <= 65535) return n;
  }
  return undefined;
}

/**
 * 出口 IP 列表「代理服务器」列：代理隧道固定展示 proxy_config.server（及端口），
 * 与 ip_address 字段解耦（域名代理时 ip_address 常为 0.0.0.0）。
 * WireGuard 展示对端 endpoint，否则回退公网 ip_address。
 */
export function egressProxyEntryDisplay(ip: EgressIP): string {
  if (ip.tunnel_type === "proxy" && ip.proxy_config) {
    const c = ip.proxy_config as Record<string, unknown>;
    const server = typeof c.server === "string" ? c.server.trim() : "";
    if (server) {
      const port = proxyServerPort(c);
      if (port !== undefined) {
        const isIPv6 = server.includes(":") && !server.includes(".");
        const host =
          isIPv6 && !server.startsWith("[") ? `[${server}]` : server;
        return `${host}:${port}`;
      }
      return server;
    }
    return "—";
  }
  if (ip.tunnel_type === "wireguard") {
    const ep = ip.wg_endpoint?.trim();
    if (ep) return ep;
    if (ip.ip_address && ip.ip_address !== "0.0.0.0") return ip.ip_address;
    return "—";
  }
  if (ip.ip_address && ip.ip_address !== "0.0.0.0") return ip.ip_address;
  return "—";
}
