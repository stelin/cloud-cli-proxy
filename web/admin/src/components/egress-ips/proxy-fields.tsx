import { useState } from "react";
import { type UseFormReturn } from "react-hook-form";
import { Eye, EyeOff } from "lucide-react";
import { toast } from "sonner";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface ProxyFieldsProps {
  form: UseFormReturn<any>;
}

// ---------------------------------------------------------------------------
// Link parsers (vmess:// vless:// ss:// trojan:// socks://)
// ---------------------------------------------------------------------------

function safeBase64Decode(str: string): string {
  const padded = str.replace(/-/g, "+").replace(/_/g, "/");
  const pad = padded.length % 4;
  return atob(pad ? padded + "=".repeat(4 - pad) : padded);
}

function parseVmessLink(link: string): Record<string, any> | null {
  try {
    const json = JSON.parse(safeBase64Decode(link.slice("vmess://".length)));
    return {
      proxy_protocol: "vmess",
      proxy_server: json.add || "",
      proxy_port: Number(json.port) || 443,
      proxy_uuid: json.id || "",
      proxy_security: json.scy || "auto",
      proxy_alter_id: Number(json.aid) || 0,
      proxy_tls: json.tls === "tls",
      proxy_server_name: json.sni || "",
      proxy_tls_insecure: false,
      proxy_tls_alpn: json.alpn || "",
      proxy_transport_type: json.net === "tcp" || !json.net ? "" : json.net,
      proxy_transport_path: json.path || "",
      proxy_transport_host: json.host || "",
      proxy_transport_service_name: "",
      _label: json.ps || "",
    };
  } catch {
    return null;
  }
}

function parseShadowsocksLink(link: string): Record<string, any> | null {
  try {
    let rest = link.slice("ss://".length);
    const hashIdx = rest.indexOf("#");
    let remark = "";
    if (hashIdx !== -1) {
      remark = decodeURIComponent(rest.slice(hashIdx + 1));
      rest = rest.slice(0, hashIdx);
    }

    const atIdx = rest.lastIndexOf("@");
    let method: string, password: string, server: string, port: number;

    if (atIdx !== -1) {
      const decoded = safeBase64Decode(rest.slice(0, atIdx));
      const colonIdx = decoded.indexOf(":");
      method = decoded.slice(0, colonIdx);
      password = decoded.slice(colonIdx + 1);
      const serverPart = rest.slice(atIdx + 1);
      const lastColon = serverPart.lastIndexOf(":");
      server = serverPart.slice(0, lastColon);
      port = Number(serverPart.slice(lastColon + 1)) || 8388;
    } else {
      const decoded = safeBase64Decode(rest);
      const match = decoded.match(/^(.+?):(.+)@(.+):(\d+)$/);
      if (!match) return null;
      method = match[1];
      password = match[2];
      server = match[3];
      port = Number(match[4]);
    }

    return {
      proxy_protocol: "shadowsocks",
      proxy_server: server,
      proxy_port: port,
      proxy_method: method,
      proxy_password: password,
      _label: remark,
    };
  } catch {
    return null;
  }
}

function parseVlessLink(link: string): Record<string, any> | null {
  try {
    const url = new URL(link);
    const params = url.searchParams;
    const security = params.get("security") || "";
    const type = params.get("type") || "";
    const isReality = security === "reality";

    return {
      proxy_protocol: "vless",
      proxy_server: url.hostname,
      proxy_port: Number(url.port) || 443,
      proxy_uuid: decodeURIComponent(url.username),
      proxy_flow: params.get("flow") || "",
      proxy_tls: security === "tls" || isReality,
      proxy_server_name: params.get("sni") || "",
      proxy_tls_insecure: params.get("allowInsecure") === "1",
      proxy_tls_alpn: params.get("alpn")
        ? decodeURIComponent(params.get("alpn")!)
        : "",
      proxy_reality: isReality,
      proxy_reality_public_key: params.get("pbk") || "",
      proxy_reality_short_id: params.get("sid") || "",
      proxy_transport_type: type === "tcp" || !type ? "" : type,
      proxy_transport_path: params.get("path")
        ? decodeURIComponent(params.get("path")!)
        : "",
      proxy_transport_host: params.get("host") || "",
      proxy_transport_service_name: params.get("serviceName") || "",
      _label: url.hash ? decodeURIComponent(url.hash.slice(1)) : "",
    };
  } catch {
    return null;
  }
}

function parseTrojanLink(link: string): Record<string, any> | null {
  try {
    const url = new URL(link);
    const params = url.searchParams;
    const type = params.get("type") || "";
    const security = params.get("security") || "tls";

    return {
      proxy_protocol: "trojan",
      proxy_server: url.hostname,
      proxy_port: Number(url.port) || 443,
      proxy_password: decodeURIComponent(url.username),
      proxy_tls: security !== "none",
      proxy_server_name: params.get("sni") || "",
      proxy_tls_insecure: params.get("allowInsecure") === "1",
      proxy_tls_alpn: params.get("alpn")
        ? decodeURIComponent(params.get("alpn")!)
        : "",
      proxy_transport_type: type === "tcp" || !type ? "" : type,
      proxy_transport_path: params.get("path")
        ? decodeURIComponent(params.get("path")!)
        : "",
      proxy_transport_host: params.get("host") || "",
      proxy_transport_service_name: params.get("serviceName") || "",
      _label: url.hash ? decodeURIComponent(url.hash.slice(1)) : "",
    };
  } catch {
    return null;
  }
}

function parseSocksLink(link: string): Record<string, any> | null {
  try {
    const url = new URL(link.replace(/^socks5?:\/\//, "http://"));
    return {
      proxy_protocol: "socks",
      proxy_server: url.hostname,
      proxy_port: Number(url.port) || 1080,
      proxy_username: decodeURIComponent(url.username || ""),
      proxy_password: decodeURIComponent(url.password || ""),
      _label: url.hash ? decodeURIComponent(url.hash.slice(1)) : "",
    };
  } catch {
    return null;
  }
}

function parseProxyLink(link: string): Record<string, any> | null {
  link = link.trim();
  if (link.startsWith("vmess://")) return parseVmessLink(link);
  if (link.startsWith("vless://")) return parseVlessLink(link);
  if (link.startsWith("ss://")) return parseShadowsocksLink(link);
  if (link.startsWith("trojan://")) return parseTrojanLink(link);
  if (link.startsWith("socks://") || link.startsWith("socks5://"))
    return parseSocksLink(link);
  return null;
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function ProxyFields({ form }: ProxyFieldsProps) {
  const editMode = form.watch("edit_mode");
  const protocol = form.watch("proxy_protocol") || "socks";
  const [importLink, setImportLink] = useState("");

  function handleImport() {
    const result = parseProxyLink(importLink.trim());
    if (!result) {
      toast.error("无法解析该链接，请检查格式是否正确");
      return;
    }
    const label = result._label;
    delete result._label;
    Object.entries(result).forEach(([k, v]) => form.setValue(k as any, v));
    if (label && !form.getValues("label")) {
      form.setValue("label", label);
    }
    setImportLink("");
    toast.success("链接导入成功");
  }

  return (
    <div className="space-y-4">
      {/* Import link */}
      <div className="flex gap-2">
        <Input
          placeholder="粘贴 vmess:// vless:// ss:// trojan:// 链接"
          value={importLink}
          onChange={(e) => setImportLink(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              handleImport();
            }
          }}
          className="text-xs"
        />
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleImport}
          disabled={!importLink.trim()}
        >
          导入
        </Button>
      </div>

      {editMode === "json" ? (
        <>
          <div className="space-y-2">
            <Label>sing-box outbound JSON</Label>
            <textarea
              className="min-h-[200px] w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              value={form.watch("proxy_config_json") || ""}
              onChange={(e) =>
                form.setValue("proxy_config_json", e.target.value)
              }
            />
            {form.formState.errors.proxy_config_json && (
              <p className="text-sm text-destructive">
                {form.formState.errors.proxy_config_json.message as string}
              </p>
            )}
          </div>
          <button
            type="button"
            className="text-xs text-muted-foreground underline underline-offset-2 hover:text-foreground"
            onClick={() => {
              const jsonStr = form.getValues("proxy_config_json");
              if (jsonStr) {
                try {
                  const parsed = JSON.parse(jsonStr);
                  const formVals = proxyConfigToFormValues(parsed);
                  Object.entries(formVals).forEach(([k, v]) =>
                    form.setValue(k as any, v),
                  );
                } catch {
                  toast.error("JSON 格式不正确，无法切换到表单模式");
                  return;
                }
              }
              form.setValue("edit_mode", "form");
            }}
          >
            切换到表单模式
          </button>
        </>
      ) : (
        <>
          <div className="space-y-2">
            <Label>协议类型</Label>
            <Select
              value={protocol}
              onValueChange={(val) => form.setValue("proxy_protocol", val)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="socks">SOCKS5</SelectItem>
                <SelectItem value="vmess">VMess</SelectItem>
                <SelectItem value="vless">VLESS</SelectItem>
                <SelectItem value="shadowsocks">Shadowsocks</SelectItem>
                <SelectItem value="trojan">Trojan</SelectItem>
                <SelectItem value="http">HTTP</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-3 gap-2">
            <div className="col-span-2 space-y-2">
              <Label htmlFor="proxy_server">服务器地址 *</Label>
              <Input
                id="proxy_server"
                placeholder="IP 或域名"
                {...form.register("proxy_server")}
              />
              {form.formState.errors.proxy_server && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.proxy_server.message as string}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="proxy_port">端口 *</Label>
              <Input
                id="proxy_port"
                type="number"
                placeholder="1080"
                {...form.register("proxy_port")}
              />
              {form.formState.errors.proxy_port && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.proxy_port.message as string}
                </p>
              )}
            </div>
          </div>

          {protocol === "socks" && <SocksFields form={form} />}
          {protocol === "vmess" && <VmessFields form={form} />}
          {protocol === "vless" && <VlessFields form={form} />}
          {protocol === "shadowsocks" && <ShadowsocksFields form={form} />}
          {protocol === "trojan" && <TrojanFields form={form} />}
          {protocol === "http" && <HttpFields form={form} />}

          <button
            type="button"
            className="text-xs text-muted-foreground underline underline-offset-2 hover:text-foreground"
            onClick={() => {
              const values = form.getValues();
              const config = formValuesToProxyConfig(values);
              form.setValue(
                "proxy_config_json",
                JSON.stringify(config, null, 2),
              );
              form.setValue("edit_mode", "json");
            }}
          >
            切换到 JSON 模式
          </button>
        </>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Shared sub-components
// ---------------------------------------------------------------------------

function PasswordField({
  form,
  name,
  label,
  required,
}: {
  form: UseFormReturn<any>;
  name: string;
  label: string;
  required?: boolean;
}) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="space-y-2">
      <Label htmlFor={name}>{label}</Label>
      <div className="relative">
        <Input
          id={name}
          type={visible ? "text" : "password"}
          placeholder={required ? undefined : "不修改则留空"}
          {...form.register(name)}
        />
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="absolute right-0 top-0 h-full px-3"
          onClick={() => setVisible(!visible)}
        >
          {visible ? (
            <EyeOff className="h-4 w-4" />
          ) : (
            <Eye className="h-4 w-4" />
          )}
        </Button>
      </div>
      {form.formState.errors[name] && (
        <p className="text-sm text-destructive">
          {form.formState.errors[name]?.message as string}
        </p>
      )}
    </div>
  );
}

function TlsFields({
  form,
  idSuffix = "",
  showReality = false,
}: {
  form: UseFormReturn<any>;
  idSuffix?: string;
  showReality?: boolean;
}) {
  const tlsEnabled = form.watch("proxy_tls");
  const realityEnabled = form.watch("proxy_reality");
  return (
    <>
      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          id={`proxy_tls${idSuffix}`}
          checked={!!tlsEnabled}
          onChange={(e) => {
            form.setValue("proxy_tls", e.target.checked);
            if (!e.target.checked) form.setValue("proxy_reality", false);
          }}
          className="rounded"
        />
        <Label htmlFor={`proxy_tls${idSuffix}`}>启用 TLS</Label>
      </div>
      {tlsEnabled && (
        <div className="space-y-3 rounded-md border border-dashed p-3">
          <div className="space-y-2">
            <Label htmlFor={`proxy_server_name${idSuffix}`}>
              SNI (Server Name)
            </Label>
            <Input
              id={`proxy_server_name${idSuffix}`}
              placeholder="留空则使用服务器地址"
              {...form.register("proxy_server_name")}
            />
          </div>
          {showReality && (
            <>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id={`proxy_reality${idSuffix}`}
                  checked={!!realityEnabled}
                  onChange={(e) =>
                    form.setValue("proxy_reality", e.target.checked)
                  }
                  className="rounded"
                />
                <Label htmlFor={`proxy_reality${idSuffix}`}>Reality</Label>
              </div>
              {realityEnabled && (
                <div className="space-y-3 rounded-md border border-dashed p-3">
                  <div className="space-y-2">
                    <Label htmlFor="proxy_reality_public_key">
                      Public Key *
                    </Label>
                    <Input
                      id="proxy_reality_public_key"
                      className="font-mono"
                      {...form.register("proxy_reality_public_key")}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="proxy_reality_short_id">Short ID</Label>
                    <Input
                      id="proxy_reality_short_id"
                      className="font-mono"
                      placeholder="可选"
                      {...form.register("proxy_reality_short_id")}
                    />
                  </div>
                </div>
              )}
            </>
          )}
          {!realityEnabled && (
            <>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id={`proxy_tls_insecure${idSuffix}`}
                  checked={!!form.watch("proxy_tls_insecure")}
                  onChange={(e) =>
                    form.setValue("proxy_tls_insecure", e.target.checked)
                  }
                  className="rounded"
                />
                <Label htmlFor={`proxy_tls_insecure${idSuffix}`}>
                  跳过证书验证
                </Label>
              </div>
              <div className="space-y-2">
                <Label htmlFor={`proxy_tls_alpn${idSuffix}`}>ALPN</Label>
                <Input
                  id={`proxy_tls_alpn${idSuffix}`}
                  placeholder="例如 h2,http/1.1"
                  {...form.register("proxy_tls_alpn")}
                />
              </div>
            </>
          )}
        </div>
      )}
    </>
  );
}

function TransportFields({ form }: { form: UseFormReturn<any> }) {
  const transportType = form.watch("proxy_transport_type") || "";
  return (
    <>
      <div className="space-y-2">
        <Label>传输层</Label>
        <Select
          value={transportType || "tcp"}
          onValueChange={(v) =>
            form.setValue("proxy_transport_type", v === "tcp" ? "" : v)
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="tcp">TCP（直连）</SelectItem>
            <SelectItem value="ws">WebSocket</SelectItem>
            <SelectItem value="grpc">gRPC</SelectItem>
            <SelectItem value="http">HTTP/2</SelectItem>
            <SelectItem value="httpupgrade">HTTPUpgrade</SelectItem>
            <SelectItem value="quic">QUIC</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {["ws", "http", "httpupgrade"].includes(transportType) && (
        <div className="space-y-3 rounded-md border border-dashed p-3">
          <div className="space-y-2">
            <Label htmlFor="proxy_transport_path">路径</Label>
            <Input
              id="proxy_transport_path"
              placeholder="/"
              {...form.register("proxy_transport_path")}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="proxy_transport_host">Host</Label>
            <Input
              id="proxy_transport_host"
              placeholder="留空则使用服务器地址"
              {...form.register("proxy_transport_host")}
            />
          </div>
        </div>
      )}
      {transportType === "grpc" && (
        <div className="space-y-2">
          <Label htmlFor="proxy_transport_service_name">Service Name</Label>
          <Input
            id="proxy_transport_service_name"
            {...form.register("proxy_transport_service_name")}
          />
        </div>
      )}
    </>
  );
}

// ---------------------------------------------------------------------------
// Protocol-specific fields
// ---------------------------------------------------------------------------

function SocksFields({ form }: { form: UseFormReturn<any> }) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="proxy_username">用户名（选填）</Label>
        <Input id="proxy_username" {...form.register("proxy_username")} />
      </div>
      <PasswordField form={form} name="proxy_password" label="密码（选填）" />
    </>
  );
}

function VmessFields({ form }: { form: UseFormReturn<any> }) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="proxy_uuid">UUID *</Label>
        <Input
          id="proxy_uuid"
          className="font-mono"
          {...form.register("proxy_uuid")}
        />
        {form.formState.errors.proxy_uuid && (
          <p className="text-sm text-destructive">
            {form.formState.errors.proxy_uuid.message as string}
          </p>
        )}
      </div>
      <div className="space-y-2">
        <Label>加密方式</Label>
        <Select
          value={form.watch("proxy_security") || "auto"}
          onValueChange={(v) => form.setValue("proxy_security", v)}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="auto">auto</SelectItem>
            <SelectItem value="none">none</SelectItem>
            <SelectItem value="zero">zero</SelectItem>
            <SelectItem value="aes-128-gcm">aes-128-gcm</SelectItem>
            <SelectItem value="chacha20-poly1305">chacha20-poly1305</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label htmlFor="proxy_alter_id">Alter ID</Label>
        <Input
          id="proxy_alter_id"
          type="number"
          defaultValue={0}
          {...form.register("proxy_alter_id")}
        />
      </div>
      <TransportFields form={form} />
      <TlsFields form={form} idSuffix="_vmess" />
    </>
  );
}

function VlessFields({ form }: { form: UseFormReturn<any> }) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="proxy_uuid">UUID *</Label>
        <Input
          id="proxy_uuid"
          className="font-mono"
          {...form.register("proxy_uuid")}
        />
        {form.formState.errors.proxy_uuid && (
          <p className="text-sm text-destructive">
            {form.formState.errors.proxy_uuid.message as string}
          </p>
        )}
      </div>
      <div className="space-y-2">
        <Label>Flow</Label>
        <Select
          value={form.watch("proxy_flow") || "none"}
          onValueChange={(v) =>
            form.setValue("proxy_flow", v === "none" ? "" : v)
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="none">无</SelectItem>
            <SelectItem value="xtls-rprx-vision">xtls-rprx-vision</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <TransportFields form={form} />
      <TlsFields form={form} idSuffix="_vless" showReality />
    </>
  );
}

function ShadowsocksFields({ form }: { form: UseFormReturn<any> }) {
  return (
    <>
      <div className="space-y-2">
        <Label>加密方式 *</Label>
        <Select
          value={form.watch("proxy_method") || ""}
          onValueChange={(v) => form.setValue("proxy_method", v)}
        >
          <SelectTrigger>
            <SelectValue placeholder="选择加密方式" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="2022-blake3-aes-128-gcm">
              2022-blake3-aes-128-gcm
            </SelectItem>
            <SelectItem value="2022-blake3-aes-256-gcm">
              2022-blake3-aes-256-gcm
            </SelectItem>
            <SelectItem value="2022-blake3-chacha20-poly1305">
              2022-blake3-chacha20-poly1305
            </SelectItem>
            <SelectItem value="aes-128-gcm">aes-128-gcm</SelectItem>
            <SelectItem value="aes-256-gcm">aes-256-gcm</SelectItem>
            <SelectItem value="chacha20-ietf-poly1305">
              chacha20-ietf-poly1305
            </SelectItem>
            <SelectItem value="xchacha20-ietf-poly1305">
              xchacha20-ietf-poly1305
            </SelectItem>
          </SelectContent>
        </Select>
        {form.formState.errors.proxy_method && (
          <p className="text-sm text-destructive">
            {form.formState.errors.proxy_method.message as string}
          </p>
        )}
      </div>
      <PasswordField
        form={form}
        name="proxy_password"
        label="密码 *"
        required
      />
    </>
  );
}

function TrojanFields({ form }: { form: UseFormReturn<any> }) {
  return (
    <>
      <PasswordField
        form={form}
        name="proxy_password"
        label="密码 *"
        required
      />
      <TransportFields form={form} />
      <TlsFields form={form} idSuffix="_trojan" />
    </>
  );
}

function HttpFields({ form }: { form: UseFormReturn<any> }) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="proxy_username">用户名（选填）</Label>
        <Input id="proxy_username" {...form.register("proxy_username")} />
      </div>
      <PasswordField form={form} name="proxy_password" label="密码（选填）" />
      <TlsFields form={form} idSuffix="_http" />
    </>
  );
}

// ---------------------------------------------------------------------------
// Config conversion (form ⇄ sing-box JSON)
// ---------------------------------------------------------------------------

export function formValuesToProxyConfig(
  values: Record<string, any>,
): Record<string, unknown> {
  const config: Record<string, unknown> = {
    type: values.proxy_protocol,
    server: values.proxy_server,
    server_port: Number(values.proxy_port),
  };

  switch (values.proxy_protocol) {
    case "socks":
      config.version = "5";
      if (values.proxy_username) config.username = values.proxy_username;
      if (values.proxy_password && values.proxy_password !== "***")
        config.password = values.proxy_password;
      break;
    case "vmess":
      config.uuid = values.proxy_uuid;
      config.security = values.proxy_security || "auto";
      config.alter_id = Number(values.proxy_alter_id) || 0;
      break;
    case "vless":
      config.uuid = values.proxy_uuid;
      if (values.proxy_flow) config.flow = values.proxy_flow;
      break;
    case "shadowsocks":
      config.method = values.proxy_method;
      if (values.proxy_password && values.proxy_password !== "***")
        config.password = values.proxy_password;
      break;
    case "trojan":
      if (values.proxy_password && values.proxy_password !== "***")
        config.password = values.proxy_password;
      break;
    case "http":
      if (values.proxy_username) config.username = values.proxy_username;
      if (values.proxy_password && values.proxy_password !== "***")
        config.password = values.proxy_password;
      break;
  }

  // TLS
  if (
    values.proxy_tls &&
    ["vmess", "vless", "trojan", "http"].includes(values.proxy_protocol)
  ) {
    const tls: Record<string, unknown> = { enabled: true };
    if (values.proxy_server_name) tls.server_name = values.proxy_server_name;
    if (values.proxy_reality) {
      tls.utls = { enabled: true, fingerprint: "chrome" };
      tls.reality = {
        enabled: true,
        public_key: values.proxy_reality_public_key || "",
        short_id: values.proxy_reality_short_id || "",
      };
    } else {
      if (values.proxy_tls_insecure) tls.insecure = true;
      if (values.proxy_tls_alpn) {
        tls.alpn = values.proxy_tls_alpn
          .split(",")
          .map((s: string) => s.trim())
          .filter(Boolean);
      }
    }
    config.tls = tls;
  }

  // Transport
  const tt = values.proxy_transport_type;
  if (
    tt &&
    tt !== "tcp" &&
    ["vmess", "vless", "trojan"].includes(values.proxy_protocol)
  ) {
    const transport: Record<string, unknown> = { type: tt };
    if (["ws", "http", "httpupgrade"].includes(tt)) {
      if (values.proxy_transport_path)
        transport.path = values.proxy_transport_path;
      if (values.proxy_transport_host) {
        transport.headers =
          tt === "ws"
            ? { Host: values.proxy_transport_host }
            : { Host: [values.proxy_transport_host] };
      }
    }
    if (tt === "grpc" && values.proxy_transport_service_name) {
      transport.service_name = values.proxy_transport_service_name;
    }
    config.transport = transport;
  }

  return config;
}

export function proxyConfigToFormValues(
  config: Record<string, unknown>,
): Record<string, unknown> {
  const protocol = config.type as string;
  const values: Record<string, unknown> = {
    proxy_protocol: protocol,
    proxy_server: (config.server as string) || "",
    proxy_port: config.server_port as number,
    proxy_username: (config.username as string) || "",
    proxy_password: (config.password as string) || "",
    proxy_tls: false,
    proxy_server_name: "",
    proxy_tls_insecure: false,
    proxy_tls_alpn: "",
    proxy_flow: "",
    proxy_reality: false,
    proxy_reality_public_key: "",
    proxy_reality_short_id: "",
    proxy_transport_type: "",
    proxy_transport_path: "",
    proxy_transport_host: "",
    proxy_transport_service_name: "",
  };

  switch (protocol) {
    case "vmess":
      values.proxy_uuid = (config.uuid as string) || "";
      values.proxy_security = (config.security as string) || "auto";
      values.proxy_alter_id = (config.alter_id as number) || 0;
      break;
    case "vless":
      values.proxy_uuid = (config.uuid as string) || "";
      values.proxy_flow = (config.flow as string) || "";
      break;
    case "shadowsocks":
      values.proxy_method = (config.method as string) || "";
      break;
  }

  // TLS
  if (config.tls && typeof config.tls === "object") {
    const tls = config.tls as Record<string, unknown>;
    values.proxy_tls = !!tls.enabled;
    values.proxy_server_name = (tls.server_name as string) || "";
    values.proxy_tls_insecure = !!tls.insecure;
    if (Array.isArray(tls.alpn)) {
      values.proxy_tls_alpn = tls.alpn.join(", ");
    }
    if (tls.reality && typeof tls.reality === "object") {
      const reality = tls.reality as Record<string, unknown>;
      values.proxy_reality = !!reality.enabled;
      values.proxy_reality_public_key =
        (reality.public_key as string) || "";
      values.proxy_reality_short_id = (reality.short_id as string) || "";
    }
  }

  // Transport
  if (config.transport && typeof config.transport === "object") {
    const transport = config.transport as Record<string, unknown>;
    values.proxy_transport_type = (transport.type as string) || "";
    values.proxy_transport_path = (transport.path as string) || "";
    if (transport.headers && typeof transport.headers === "object") {
      const headers = transport.headers as Record<string, unknown>;
      const host = headers.Host;
      if (Array.isArray(host)) {
        values.proxy_transport_host = (host[0] as string) || "";
      } else if (typeof host === "string") {
        values.proxy_transport_host = host;
      }
    }
    values.proxy_transport_service_name =
      (transport.service_name as string) || "";
  }

  return values;
}
