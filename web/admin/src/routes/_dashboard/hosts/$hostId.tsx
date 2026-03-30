import { useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Check, Copy, KeyRound, Monitor, Terminal } from "lucide-react";
import { toast } from "sonner";
import { getToken } from "@/lib/auth";
import { useHostDetail } from "@/hooks/use-hosts";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { BindingManager } from "@/components/hosts/binding-manager";
import { HostLifecycleActions } from "@/components/hosts/host-lifecycle-actions";
import { RotatePasswordDialog } from "@/components/users/rotate-password-dialog";
import { RotateHostSSHPasswordDialog } from "@/components/hosts/rotate-host-ssh-password-dialog";

export const Route = createFileRoute("/_dashboard/hosts/$hostId")({
  component: HostDetailPage,
});

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

const statusConfig: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  running: { label: "运行中", variant: "default" },
  stopped: { label: "已停止", variant: "secondary" },
  pending: { label: "等待中", variant: "outline" },
  failed: { label: "失败", variant: "destructive" },
};

function HostDetailPage() {
  const { hostId } = Route.useParams();
  const { data, isLoading } = useHostDetail(hostId);
  const [rotateLoginOpen, setRotateLoginOpen] = useState(false);
  const [rotateSSHOpen, setRotateSSHOpen] = useState(false);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-40 animate-pulse rounded bg-muted" />
      </div>
    );
  }

  if (!data) {
    return (
      <div className="py-12 text-center text-muted-foreground">
        主机不存在
      </div>
    );
  }

  const { host, user, bindings } = data;
  const sc = statusConfig[host.status] ?? {
    label: host.status,
    variant: "outline" as const,
  };

  function openVNC() {
    const token = getToken();
    const wsPath = encodeURIComponent(`v1/admin/hosts/${host.id}/vnc/`);
    window.open(
      `/v1/admin/hosts/${host.id}/vnc/vnc.html?autoconnect=true&resize=remote&path=${wsPath}&token=${token}`,
      "_blank",
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to="/hosts"
          className="text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <h1 className="text-2xl font-bold">主机详情</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>基本信息</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
            <div>
              <dt className="text-muted-foreground">主机 ID</dt>
              <dd className="font-mono">{host.id}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">主机短 ID</dt>
              <dd className="font-mono">{host.short_id || "—"}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">状态</dt>
              <dd>
                <Badge variant={sc.variant}>{sc.label}</Badge>
              </dd>
            </div>
            <div>
              <dt className="text-muted-foreground">所属用户</dt>
              <dd>
                <Link
                  to="/users/$userId"
                  params={{ userId: user.id }}
                  className="text-primary hover:underline"
                >
                  {user.username}
                </Link>
              </dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Slot Key</dt>
              <dd>{host.slot_key}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">主机名</dt>
              <dd>{host.hostname || "—"}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">镜像模板</dt>
              <dd className="font-mono text-xs">{host.template_image_ref}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">时区</dt>
              <dd>{host.timezone || "—"}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">创建时间</dt>
              <dd>{formatDate(host.created_at)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">更新时间</dt>
              <dd>{formatDate(host.updated_at)}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      {data.connection_info && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Terminal className="h-5 w-5" />
              连接方式
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                一键连接（curl 入口）：
              </p>
              <CopyableCommand command={data.connection_info.curl_command} />
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                SSH 直连（需要这台主机的 SSH 密码）：
              </p>
              <CopyableCommand command={data.connection_info.ssh_command} />
            </div>
            {data.connection_info.vnc_url && (
              <div className="space-y-2">
                <p className="text-sm text-muted-foreground">
                  VNC 登录入口：
                </p>
                <div className="flex flex-wrap gap-2">
                  <CopyableCommand command={data.connection_info.vnc_url} />
                  <Button type="button" variant="outline" onClick={openVNC}>
                    <Monitor className="mr-2 h-4 w-4" />
                    打开桌面
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>出口 IP 绑定</CardTitle>
          </CardHeader>
          <CardContent>
            <BindingManager
              hostId={hostId}
              hostStatus={host.status}
              bindings={bindings}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>生命周期与密码操作</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <HostLifecycleActions hostId={hostId} hostStatus={host.status} />
            <Separator className="my-4" />
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setRotateSSHOpen(true)}
              >
                <KeyRound className="mr-2 h-4 w-4" />
                重置主机 SSH 密码
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() => setRotateLoginOpen(true)}
              >
                <KeyRound className="mr-2 h-4 w-4" />
                轮换用户登录密码
              </Button>
              <Button type="button" variant="outline" onClick={openVNC}>
                <Monitor className="mr-2 h-4 w-4" />
                打开 VNC 桌面
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              操作提交后将异步执行，请在任务列表中查看进度。
            </p>
          </CardContent>
        </Card>
      </div>

      <RotatePasswordDialog
        userId={user.id}
        open={rotateLoginOpen}
        onOpenChange={setRotateLoginOpen}
      />
      <RotateHostSSHPasswordDialog
        hostId={host.id}
        open={rotateSSHOpen}
        onOpenChange={setRotateSSHOpen}
      />
    </div>
  );
}

function CopyableCommand({ command }: { command: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(command).then(() => {
      setCopied(true);
      toast.success("已复制到剪贴板");
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div className="flex items-center gap-2 rounded-md border bg-muted p-3">
      <code className="flex-1 break-all font-mono text-sm">{command}</code>
      <Button variant="ghost" size="icon" className="shrink-0" onClick={handleCopy}>
        {copied ? (
          <Check className="h-4 w-4 text-green-600" />
        ) : (
          <Copy className="h-4 w-4" />
        )}
      </Button>
    </div>
  );
}
