import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import {
  Users,
  Server,
  Globe,
  TrendingUp,
  ArrowRight,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useEvents, eventTypeLabel } from "@/hooks/use-events";

interface DashboardStats {
  active_users: number;
  running_hosts: number;
  available_ips: number;
}

export const Route = createFileRoute("/_dashboard/")({
  component: DashboardPage,
});

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  });
}

function DashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["dashboard-stats"],
    queryFn: () => apiFetch<DashboardStats>("/dashboard/stats"),
  });

  const { data: eventsData, isLoading: eventsLoading } = useEvents({
    limit: 5,
  });
  const events = eventsData?.events ?? [];

  const cards = [
    {
      title: "活跃用户",
      value: data?.active_users,
      icon: Users,
      gradient: "from-blue-500/10 to-blue-600/5",
      iconBg: "bg-blue-500/10",
      iconColor: "text-blue-600",
      link: "/users",
    },
    {
      title: "运行中主机",
      value: data?.running_hosts,
      icon: Server,
      gradient: "from-emerald-500/10 to-emerald-600/5",
      iconBg: "bg-emerald-500/10",
      iconColor: "text-emerald-600",
      link: "/hosts",
    },
    {
      title: "可用出口 IP",
      value: data?.available_ips,
      icon: Globe,
      gradient: "from-violet-500/10 to-violet-600/5",
      iconBg: "bg-violet-500/10",
      iconColor: "text-violet-600",
      link: "/egress-ips",
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">仪表板</h1>
          <p className="text-sm text-muted-foreground mt-1">系统运行状态概览</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <div className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
            系统正常
          </div>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {cards.map((card) => (
          <Link key={card.title} to={card.link}>
            <Card className="group relative overflow-hidden transition-all duration-200 hover:shadow-md hover:-translate-y-0.5 cursor-pointer">
              <div className={`absolute inset-0 bg-linear-to-br ${card.gradient} opacity-0 group-hover:opacity-100 transition-opacity`} />
              <CardHeader className="relative flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  {card.title}
                </CardTitle>
                <div className={`flex h-9 w-9 items-center justify-center rounded-xl ${card.iconBg}`}>
                  <card.icon className={`h-4.5 w-4.5 ${card.iconColor}`} />
                </div>
              </CardHeader>
              <CardContent className="relative">
                {isLoading ? (
                  <div className="h-9 w-20 animate-pulse rounded-lg bg-muted" />
                ) : (
                  <div className="flex items-end gap-2">
                    <p className="text-3xl font-bold tracking-tight">{card.value ?? "–"}</p>
                    <TrendingUp className="h-4 w-4 mb-1.5 text-muted-foreground/40" />
                  </div>
                )}
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle className="text-base font-semibold">最近事件</CardTitle>
            <p className="text-xs text-muted-foreground mt-0.5">系统活动记录</p>
          </div>
          <Link
            to="/events"
            className="flex items-center gap-1 text-xs text-primary hover:underline font-medium"
          >
            查看全部
            <ArrowRight className="h-3 w-3" />
          </Link>
        </CardHeader>
        <CardContent>
          {eventsLoading ? (
            <div className="space-y-4">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3">
                  <div className="h-4 w-12 animate-pulse rounded bg-muted" />
                  <div className="h-4 w-16 animate-pulse rounded bg-muted" />
                  <div className="h-4 flex-1 animate-pulse rounded bg-muted" />
                </div>
              ))}
            </div>
          ) : events.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
              <p className="text-sm">暂无事件</p>
            </div>
          ) : (
            <div className="space-y-3">
              {events.map((event) => (
                <div
                  key={event.id}
                  className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors hover:bg-muted/50"
                >
                  <span className="text-xs font-mono text-muted-foreground whitespace-nowrap tabular-nums">
                    {formatTime(event.created_at)}
                  </span>
                  <Badge
                    variant={
                      event.level === "info" ? "secondary" : "destructive"
                    }
                    className="text-[10px] font-medium"
                  >
                    {eventTypeLabel(event.type)}
                  </Badge>
                  <span className="truncate text-foreground/80">{event.message}</span>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
