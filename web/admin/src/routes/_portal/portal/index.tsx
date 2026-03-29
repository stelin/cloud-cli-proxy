import { createFileRoute, Link } from "@tanstack/react-router";
import { Globe } from "lucide-react";
import { useMyHosts } from "@/hooks/use-portal-hosts";
import type { PortalHost } from "@/hooks/use-portal-hosts";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

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

function PortalHostList() {
  const { data, isLoading } = useMyHosts();
  const hosts = data?.hosts ?? [];

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
    </div>
  );
}
