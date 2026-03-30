import { createFileRoute } from "@tanstack/react-router";
import { ListTodo } from "lucide-react";
import { useTasks } from "@/hooks/use-tasks";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/layout/page-header";
import { DataTableShell } from "@/components/layout/data-table-shell";
import { EmptyState } from "@/components/layout/empty-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export const Route = createFileRoute("/_dashboard/tasks/")({
  component: TasksPage,
});

function formatDate(dateStr: string) {
  const d = new Date(dateStr);
  return d.toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

const statusConfig: Record<
  string,
  { label: string; variant: "default" | "secondary" | "destructive" | "outline" }
> = {
  pending: { label: "等待中", variant: "outline" },
  running: { label: "运行中", variant: "default" },
  succeeded: { label: "成功", variant: "default" },
  failed: { label: "失败", variant: "destructive" },
  canceled: { label: "已取消", variant: "secondary" },
};

function TasksPage() {
  const { data, isLoading } = useTasks();
  const tasks = data?.tasks ?? [];

  return (
    <div className="space-y-6">
      <PageHeader
        title="任务列表"
        description="异步任务与主机操作编排的执行进度，每 5 秒自动刷新"
      />

      <DataTableShell>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>任务 ID</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>请求方</TableHead>
              <TableHead>错误信息</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead>更新时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 7 }).map((_, j) => (
                    <TableCell key={j}>
                      <div className="h-4 w-20 animate-pulse rounded bg-muted" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : tasks.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="p-0">
                  <EmptyState
                    icon={ListTodo}
                    title="暂无任务"
                    description="创建、启动或重建主机等操作会在此生成任务记录"
                  />
                </TableCell>
              </TableRow>
            ) : (
              tasks.map((task) => {
                const sc = statusConfig[task.status] ?? {
                  label: task.status,
                  variant: "outline" as const,
                };
                return (
                  <TableRow key={task.task_id}>
                    <TableCell className="font-mono text-sm">
                      {task.task_id.slice(0, 8)}…
                    </TableCell>
                    <TableCell>{task.kind}</TableCell>
                    <TableCell>
                      <Badge variant={sc.variant}>{sc.label}</Badge>
                    </TableCell>
                    <TableCell>{task.requested_by ?? "—"}</TableCell>
                    <TableCell className="max-w-[200px] truncate text-sm text-destructive">
                      {task.status === "failed" ? (task.last_error_summary || "—") : "—"}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {task.created_at ? formatDate(task.created_at) : "—"}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDate(task.updated_at)}
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </DataTableShell>
    </div>
  );
}
