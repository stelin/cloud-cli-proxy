import { useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { ChevronDown, ChevronRight, RotateCcw, ScrollText, Calendar } from "lucide-react";
import { useEvents, eventTypeLabel, type EventItem } from "@/hooks/use-events";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PageHeader } from "@/components/layout/page-header";
import { DataTableShell } from "@/components/layout/data-table-shell";
import { EmptyState } from "@/components/layout/empty-state";

export const Route = createFileRoute("/_dashboard/events/")({
  component: EventsPage,
});

const PAGE_SIZE = 50;

const ALL_EVENT_TYPES = "__all__";

const eventTypes = [
  { value: ALL_EVENT_TYPES, label: "全部类型" },
  { value: "auth.success", label: "认证成功" },
  { value: "auth.failed", label: "认证失败" },
  { value: "user.expired", label: "用户过期" },
  { value: "host.stop.expired", label: "过期主机停止" },
  { value: "admin.user.created", label: "创建用户" },
  { value: "admin.user.updated", label: "修改用户" },
  { value: "admin.user.deleted", label: "删除用户" },
  { value: "admin.user.password_rotated", label: "轮换密码" },
  { value: "admin.binding.created", label: "创建绑定" },
  { value: "admin.binding.deleted", label: "删除绑定" },
  { value: "admin.host.action", label: "主机操作" },
  { value: "reconcile.host.drift", label: "主机漂移" },
  { value: "reconcile.task.stale", label: "陈旧任务" },
] as const;

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function EventsPage() {
  const [typeFilter, setTypeFilter] = useState<string>(ALL_EVENT_TYPES);
  const [sinceDate, setSinceDate] = useState("");
  const [untilDate, setUntilDate] = useState("");
  const [offset, setOffset] = useState(0);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const apiType =
    typeFilter === ALL_EVENT_TYPES || !typeFilter ? undefined : typeFilter;

  const { data, isLoading } = useEvents({
    type: apiType,
    since: sinceDate || undefined,
    until: untilDate || undefined,
    limit: PAGE_SIZE,
    offset,
  });

  const events = data?.events ?? [];
  const total = data?.total ?? 0;

  function handleResetFilter() {
    setTypeFilter(ALL_EVENT_TYPES);
    setSinceDate("");
    setUntilDate("");
    setOffset(0);
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="事件日志"
        description="系统审计与运维事件，按类型筛选；每 15 秒自动刷新"
      />

      <div className="flex flex-wrap items-center gap-3">
        <Select
          value={typeFilter}
          onValueChange={(v) => {
            setTypeFilter(v);
            setOffset(0);
          }}
        >
          <SelectTrigger className="w-[220px]">
            <SelectValue placeholder="全部类型" />
          </SelectTrigger>
          <SelectContent>
            {eventTypes.map((t) => (
              <SelectItem key={t.value} value={t.value}>
                {t.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="flex items-center gap-2">
          <Calendar className="h-4 w-4 text-muted-foreground shrink-0" />
          <Input
            type="datetime-local"
            value={sinceDate}
            onChange={(e) => { setSinceDate(e.target.value); setOffset(0); }}
            placeholder="开始时间"
            className="w-auto"
            aria-label="开始时间"
          />
          <span className="text-muted-foreground text-sm">-</span>
          <Input
            type="datetime-local"
            value={untilDate}
            onChange={(e) => { setUntilDate(e.target.value); setOffset(0); }}
            placeholder="结束时间"
            className="w-auto"
            aria-label="结束时间"
          />
        </div>
        {(typeFilter !== ALL_EVENT_TYPES || sinceDate || untilDate) && (
          <Button variant="ghost" size="sm" onClick={handleResetFilter}>
            <RotateCcw className="mr-1 h-3 w-3" />
            重置
          </Button>
        )}
      </div>

      <DataTableShell>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[30px]" />
              <TableHead>时间</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>消息</TableHead>
              <TableHead>用户</TableHead>
              <TableHead>主机</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 6 }).map((_, j) => (
                    <TableCell key={j}>
                      <div className="h-4 w-20 animate-pulse rounded bg-muted" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : events.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="p-0">
                  <EmptyState
                    icon={ScrollText}
                    title="暂无事件"
                    description={
                      typeFilter !== ALL_EVENT_TYPES
                        ? "当前筛选条件下没有记录，可重置类型或翻页查看"
                        : "系统尚未记录事件，或列表为空"
                    }
                    action={
                      typeFilter !== ALL_EVENT_TYPES ? (
                        <Button variant="outline" size="sm" onClick={handleResetFilter}>
                          <RotateCcw className="mr-1 h-3 w-3" />
                          重置筛选
                        </Button>
                      ) : undefined
                    }
                  />
                </TableCell>
              </TableRow>
            ) : (
              events.map((event) => (
                <EventRow
                  key={event.id}
                  event={event}
                  expanded={expandedId === event.id}
                  onToggle={() =>
                    setExpandedId(expandedId === event.id ? null : event.id)
                  }
                />
              ))
            )}
          </TableBody>
        </Table>
      </DataTableShell>

      {total > 0 && (
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <span>
            第 {offset + 1}–{Math.min(offset + events.length, total)} 条，共{" "}
            {total} 条
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={offset + PAGE_SIZE >= total}
              onClick={() => setOffset(offset + PAGE_SIZE)}
            >
              下一页
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

function EventRow({
  event,
  expanded,
  onToggle,
}: {
  event: EventItem;
  expanded: boolean;
  onToggle: () => void;
}) {
  const hasMetadata = Object.keys(event.metadata).length > 0;

  return (
    <>
      <TableRow
        className={hasMetadata ? "cursor-pointer" : ""}
        onClick={hasMetadata ? onToggle : undefined}
      >
        <TableCell className="w-[30px] px-2">
          {hasMetadata &&
            (expanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            ))}
        </TableCell>
        <TableCell className="whitespace-nowrap text-muted-foreground">
          {formatDate(event.created_at)}
        </TableCell>
        <TableCell>
          <Badge
            variant={event.level === "info" ? "secondary" : "destructive"}
          >
            {eventTypeLabel(event.type)}
          </Badge>
        </TableCell>
        <TableCell className="max-w-[300px] truncate">{event.message}</TableCell>
        <TableCell className="font-mono text-xs text-muted-foreground">
          {event.user_id ? event.user_id.slice(0, 8) : "—"}
        </TableCell>
        <TableCell className="font-mono text-xs text-muted-foreground">
          {event.host_id ? event.host_id.slice(0, 8) : "—"}
        </TableCell>
      </TableRow>
      {expanded && hasMetadata && (
        <TableRow>
          <TableCell colSpan={6} className="bg-muted/50 px-8 py-3">
            <div className="space-y-1">
              {Object.entries(event.metadata).map(([key, value]) => (
                <div key={key} className="flex gap-2 text-sm">
                  <span className="font-mono text-muted-foreground">
                    {key}:
                  </span>
                  <span>{String(value)}</span>
                </div>
              ))}
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}
