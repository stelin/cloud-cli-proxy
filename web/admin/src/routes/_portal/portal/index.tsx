import { useState, type FormEvent } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Globe } from "lucide-react";
import { toast } from "sonner";
import { useMyHosts } from "@/hooks/use-portal-hosts";
import type { PortalHost } from "@/hooks/use-portal-hosts";
import {
  useChangeLoginPassword,
  useChangeSSHPassword,
} from "@/hooks/use-portal-password";
import { ApiError } from "@/lib/api";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const Route = createFileRoute("/_portal/portal/")({
  component: PortalHostList,
});

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
}

const statusStyles: Record<string, { label: string; className: string }> = {
  running: { label: "运行中", className: "bg-green-100 text-green-700" },
  stopped: { label: "已停止", className: "bg-gray-100 text-gray-700" },
  rebuilding: { label: "重建中", className: "bg-yellow-100 text-yellow-700" },
  pending: { label: "等待中", className: "bg-blue-100 text-blue-700" },
};

function StatusBadge({ status }: { status: string }) {
  const style = statusStyles[status] ?? {
    label: status,
    className: "bg-gray-100 text-gray-700",
  };
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${style.className}`}
    >
      {style.label}
    </span>
  );
}

function HostCard({ host }: { host: PortalHost }) {
  return (
    <Link
      to="/portal/hosts/$hostId"
      params={{ hostId: host.id }}
      className="block transition-shadow hover:shadow-md"
    >
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-base font-semibold">
            {host.hostname || "未命名主机"}
          </CardTitle>
          <StatusBadge status={host.status} />
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <Globe className="h-4 w-4" />
            <span className="font-mono">{host.egress_ip || "未分配"}</span>
          </div>
          <div className="text-xs text-muted-foreground">
            创建于 {formatDate(host.created_at)}
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}

function parseApiError(err: unknown): string {
  if (err instanceof ApiError) {
    try {
      const j = JSON.parse(err.message) as { error?: string };
      if (j.error) return j.error;
    } catch {
      return err.message;
    }
  }
  return "请求失败";
}

function PortalHostList() {
  const { data, isLoading } = useMyHosts();
  const hosts = data?.hosts ?? [];
  const changeLogin = useChangeLoginPassword();
  const changeSSH = useChangeSSHPassword();
  const [loginOld, setLoginOld] = useState("");
  const [loginNew, setLoginNew] = useState("");
  const [loginConfirm, setLoginConfirm] = useState("");
  const [sshLoginPw, setSshLoginPw] = useState("");
  const [sshNew, setSshNew] = useState("");
  const [sshConfirm, setSshConfirm] = useState("");

  function submitLoginPassword(e: FormEvent) {
    e.preventDefault();
    if (loginNew.length < 8) {
      toast.error("新密码至少 8 个字符");
      return;
    }
    if (loginNew !== loginConfirm) {
      toast.error("两次输入的新密码不一致");
      return;
    }
    changeLogin.mutate(
      { old_password: loginOld, new_password: loginNew },
      {
        onSuccess: () => {
          toast.success("登录密码已更新");
          setLoginOld("");
          setLoginNew("");
          setLoginConfirm("");
        },
        onError: (err) => toast.error(parseApiError(err)),
      },
    );
  }

  function submitSSHPassword(e: FormEvent) {
    e.preventDefault();
    if (sshNew.length < 6) {
      toast.error("新 SSH 密码至少 6 个字符");
      return;
    }
    if (sshNew !== sshConfirm) {
      toast.error("两次输入的新 SSH 密码不一致");
      return;
    }
    changeSSH.mutate(
      { old_password: sshLoginPw, new_ssh_password: sshNew },
      {
        onSuccess: () => {
          toast.success("SSH 密码已更新");
          setSshLoginPw("");
          setSshNew("");
          setSshConfirm("");
        },
        onError: (err) => toast.error(parseApiError(err)),
      },
    );
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">我的主机</h1>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <div className="h-5 w-32 animate-pulse rounded bg-muted" />
              </CardHeader>
              <CardContent className="space-y-2">
                <div className="h-4 w-24 animate-pulse rounded bg-muted" />
                <div className="h-3 w-20 animate-pulse rounded bg-muted" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">我的主机</h1>
      {hosts.length === 0 ? (
        <div className="rounded-lg border border-dashed p-12 text-center">
          <p className="text-muted-foreground">暂无主机</p>
          <p className="mt-1 text-sm text-muted-foreground">
            请联系管理员为您创建主机
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {hosts.map((host) => (
            <HostCard key={host.id} host={host} />
          ))}
        </div>
      )}

      <div className="mt-10 grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>修改登录密码</CardTitle>
            <CardDescription>
              用于网页登录与 curl 入口，与 SSH 密码不同。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={submitLoginPassword} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="portal-login-old">当前密码</Label>
                <Input
                  id="portal-login-old"
                  type="password"
                  autoComplete="current-password"
                  value={loginOld}
                  onChange={(e) => setLoginOld(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="portal-login-new">新密码</Label>
                <Input
                  id="portal-login-new"
                  type="password"
                  autoComplete="new-password"
                  value={loginNew}
                  onChange={(e) => setLoginNew(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="portal-login-confirm">确认新密码</Label>
                <Input
                  id="portal-login-confirm"
                  type="password"
                  autoComplete="new-password"
                  value={loginConfirm}
                  onChange={(e) => setLoginConfirm(e.target.value)}
                />
              </div>
              <Button
                type="submit"
                disabled={
                  changeLogin.isPending ||
                  !loginOld ||
                  !loginNew ||
                  !loginConfirm
                }
              >
                {changeLogin.isPending ? "保存中…" : "更新登录密码"}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>修改 SSH 密码</CardTitle>
            <CardDescription>
              用于 SSH 连接容器；需用当前登录密码验证身份。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={submitSSHPassword} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="portal-ssh-login">登录密码（验证身份）</Label>
                <Input
                  id="portal-ssh-login"
                  type="password"
                  autoComplete="current-password"
                  value={sshLoginPw}
                  onChange={(e) => setSshLoginPw(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="portal-ssh-new">新 SSH 密码</Label>
                <Input
                  id="portal-ssh-new"
                  type="password"
                  autoComplete="new-password"
                  value={sshNew}
                  onChange={(e) => setSshNew(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="portal-ssh-confirm">确认新 SSH 密码</Label>
                <Input
                  id="portal-ssh-confirm"
                  type="password"
                  autoComplete="new-password"
                  value={sshConfirm}
                  onChange={(e) => setSshConfirm(e.target.value)}
                />
              </div>
              <Button
                type="submit"
                disabled={
                  changeSSH.isPending ||
                  !sshLoginPw ||
                  !sshNew ||
                  !sshConfirm
                }
              >
                {changeSSH.isPending ? "保存中…" : "更新 SSH 密码"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
