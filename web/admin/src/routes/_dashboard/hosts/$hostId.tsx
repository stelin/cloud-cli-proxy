import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { useHostDetail } from "@/hooks/use-hosts";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { BindingManager } from "@/components/hosts/binding-manager";
import { HostLifecycleActions } from "@/components/hosts/host-lifecycle-actions";

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
              <dt className="text-muted-foreground">镜像模板</dt>
              <dd className="font-mono text-xs">{host.template_image_ref}</dd>
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
            <CardTitle>生命周期操作</CardTitle>
          </CardHeader>
          <CardContent>
            <HostLifecycleActions hostId={hostId} hostStatus={host.status} />
            <Separator className="my-4" />
            <p className="text-xs text-muted-foreground">
              操作提交后将异步执行，请在任务列表中查看进度。
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
