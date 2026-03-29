import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import { Users, Server, Globe } from "lucide-react";
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
      color: "text-blue-600",
    },
    {
      title: "运行中主机",
      value: data?.running_hosts,
      icon: Server,
      color: "text-green-600",
    },
    {
      title: "可用出口 IP",
      value: data?.available_ips,
      icon: Globe,
      color: "text-purple-600",
    },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">仪表板</h1>
      <div className="grid gap-4 md:grid-cols-3">
        {cards.map((card) => (
          <Card key={card.title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">
                {card.title}
              </CardTitle>
              <card.icon className={`h-5 w-5 ${card.color}`} />
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="h-8 w-16 animate-pulse rounded bg-muted" />
              ) : (
                <p className="text-3xl font-bold">{card.value ?? "–"}</p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">最近事件</CardTitle>
          <Link
            to="/events"
            className="text-xs text-muted-foreground hover:underline"
          >
            查看全部
          </Link>
        </CardHeader>
        <CardContent>
          {eventsLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-4 animate-pulse rounded bg-muted" />
              ))}
            </div>
          ) : events.length === 0 ? (
            <p className="text-sm text-muted-foreground">暂无事件</p>
          ) : (
            <div className="space-y-3">
              {events.map((event) => (
                <div
                  key={event.id}
                  className="flex items-center gap-3 text-sm"
                >
                  <span className="text-xs text-muted-foreground whitespace-nowrap">
                    {formatTime(event.created_at)}
                  </span>
                  <Badge
                    variant={
                      event.level === "info" ? "secondary" : "destructive"
                    }
                    className="text-xs"
                  >
                    {eventTypeLabel(event.type)}
                  </Badge>
                  <span className="truncate">{event.message}</span>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
