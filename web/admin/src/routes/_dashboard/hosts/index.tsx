import { useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  MoreHorizontal,
  Eye,
  Plus,
  Trash2,
  Monitor,
  Play,
  Square,
  RotateCcw,
  Globe,
} from "lucide-react";
import { toast } from "sonner";
import { useHosts, useDeleteHost, useHostAction } from "@/hooks/use-hosts";
import { useTasks } from "@/hooks/use-tasks";
import { CreateHostDialog } from "@/components/hosts/create-host-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export const Route = createFileRoute("/_dashboard/hosts/")({
  component: HostsPage,
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
  {
    label: string;
    variant: "default" | "secondary" | "destructive" | "outline";
  }
> = {
  running: { label: "运行中", variant: "default" },
  stopped: { label: "已停止", variant: "secondary" },
  pending: { label: "等待中", variant: "outline" },
  failed: { label: "失败", variant: "destructive" },
};

const dockerStatusMap: Record<string, { label: string; color: string }> = {
  running: { label: "运行中", color: "text-green-600" },
  exited: { label: "已退出", color: "text-red-500" },
  created: { label: "已创建", color: "text-yellow-600" },
  paused: { label: "已暂停", color: "text-yellow-600" },
  restarting: { label: "重启中", color: "text-blue-500" },
  dead: { label: "已死亡", color: "text-red-700" },
  "not found": { label: "未创建", color: "text-muted-foreground" },
};

function HostsPage() {
  const { data, isLoading } = useHosts();
  const { data: tasksData } = useTasks();
  const hosts = data?.hosts ?? [];
  const tasks = tasksData?.tasks ?? [];
  const deleteMutation = useDeleteHost();
  const hostAction = useHostAction();
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{
    id: string;
    username: string;
    status: string;
  } | null>(null);

  function getLatestTask(hostId: string) {
    return tasks.find((t) => t.host_id === hostId);
  }

  function handleAction(
    hostId: string,
    action: "start" | "stop" | "rebuild",
    label: string,
  ) {
    hostAction.mutate(
      { hostId, action },
      {
        onSuccess: () => toast.success(`${label}指令已发送`),
        onError: () => toast.error(`${label}失败`),
      },
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">主机管理</h1>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          新建主机
        </Button>
      </div>

      <CreateHostDialog open={createOpen} onOpenChange={setCreateOpen} />

      <div className="rounded-md border bg-background">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>主机名</TableHead>
              <TableHead>所属用户</TableHead>
              <TableHead>出口 IP</TableHead>
              <TableHead>DB 状态</TableHead>
              <TableHead>容器状态</TableHead>
              <TableHead>最新任务</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead className="w-[140px]">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 8 }).map((_, j) => (
                    <TableCell key={j}>
                      <div className="h-4 w-20 animate-pulse rounded bg-muted" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : hosts.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={8}
                  className="h-24 text-center text-muted-foreground"
                >
                  暂无主机
                </TableCell>
              </TableRow>
            ) : (
              hosts.map((host) => {
                const sc = statusConfig[host.status] ?? {
                  label: host.status,
                  variant: "outline" as const,
                };
                const ds = dockerStatusMap[host.docker_status] ?? {
                  label: host.docker_status || "未知",
                  color: "text-muted-foreground",
                };
                const latestTask = getLatestTask(host.id);
                const isRunning = host.docker_status === "running";
                const isStopped =
                  host.docker_status === "exited" ||
                  host.docker_status === "not found" ||
                  host.status === "stopped";

                return (
                  <TableRow key={host.id}>
                    <TableCell>
                      <Link
                        to="/hosts/$hostId"
                        params={{ hostId: host.id }}
                        className="font-mono text-sm text-primary hover:underline"
                      >
                        {host.hostname || host.id.slice(0, 8) + "…"}
                      </Link>
                    </TableCell>
                    <TableCell>{host.username}</TableCell>
                    <TableCell>
                      {host.egress_ip_label ? (
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="inline-flex items-center gap-1 text-sm">
                                <Globe className="h-3.5 w-3.5 text-muted-foreground" />
                                {host.egress_ip_label}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent>
                              {host.egress_ip_address}
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      ) : (
                        <span className="text-sm text-muted-foreground">
                          未绑定
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant={sc.variant}>{sc.label}</Badge>
                    </TableCell>
                    <TableCell>
                      <span className={`text-sm font-medium ${ds.color}`}>
                        {ds.label}
                      </span>
                    </TableCell>
                    <TableCell>
                      <TaskStatusCell task={latestTask} />
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(host.updated_at)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-0.5">
                        {isStopped && (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-8 w-8"
                                  disabled={hostAction.isPending}
                                  onClick={() =>
                                    handleAction(host.id, "start", "启动")
                                  }
                                >
                                  <Play className="h-4 w-4 text-green-600" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>启动</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                        {isRunning && (
                          <>
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8"
                                    disabled={hostAction.isPending}
                                    onClick={() =>
                                      handleAction(host.id, "stop", "停止")
                                    }
                                  >
                                    <Square className="h-4 w-4 text-red-500" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>停止</TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8"
                                    disabled={hostAction.isPending}
                                    onClick={() =>
                                      handleAction(host.id, "rebuild", "重建")
                                    }
                                  >
                                    <RotateCcw className="h-4 w-4 text-blue-500" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>重建</TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8"
                                    onClick={() => {
                                      const token =
                                        localStorage.getItem("admin_token");
                                      const wsPath = encodeURIComponent(
                                        `v1/admin/hosts/${host.id}/vnc/`,
                                      );
                                      window.open(
                                        `/v1/admin/hosts/${host.id}/vnc/vnc.html?autoconnect=true&resize=remote&path=${wsPath}&token=${token}`,
                                        "_blank",
                                      );
                                    }}
                                  >
                                    <Monitor className="h-4 w-4" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>浏览器桌面</TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </>
                        )}
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                            >
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem asChild>
                              <Link
                                to="/hosts/$hostId"
                                params={{ hostId: host.id }}
                              >
                                <Eye />
                                查看详情
                              </Link>
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              variant="destructive"
                              onClick={() =>
                                setDeleteTarget({
                                  id: host.id,
                                  username: host.username,
                                  status: host.status,
                                })
                              }
                            >
                              <Trash2 />
                              删除
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {deleteTarget?.status === "running"
                ? "强制删除运行中的主机？"
                : "确认删除主机？"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              将移除用户 {deleteTarget?.username}{" "}
              的主机容器和数据库记录。此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => {
                if (!deleteTarget) return;
                deleteMutation.mutate(
                  {
                    hostId: deleteTarget.id,
                    force: deleteTarget.status === "running",
                  },
                  {
                    onSuccess: () => {
                      toast.success("主机已删除");
                      setDeleteTarget(null);
                    },
                    onError: () => {
                      toast.error("删除失败");
                      setDeleteTarget(null);
                    },
                  },
                );
              }}
            >
              {deleteTarget?.status === "running" ? "强制删除" : "确认删除"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function TaskStatusCell({
  task,
}: {
  task?:
    | (ReturnType<typeof useTasks>["data"] extends
        | { tasks: (infer T)[] }
        | undefined
        ? T
        : never)
    | undefined;
}) {
  if (!task) return <span className="text-sm text-muted-foreground">—</span>;

  const taskKindLabels: Record<string, string> = {
    create_host: "创建",
    start_host: "启动",
    stop_host: "停止",
    rebuild_host: "重建",
  };

  const taskStatusLabels: Record<
    string,
    { label: string; className: string }
  > = {
    pending: { label: "排队中", className: "text-muted-foreground" },
    running: { label: "执行中", className: "text-primary" },
    succeeded: { label: "成功", className: "text-green-600" },
    failed: { label: "失败", className: "text-destructive" },
    canceled: { label: "已取消", className: "text-muted-foreground" },
  };

  const kind = taskKindLabels[task.kind] ?? task.kind;
  const status = taskStatusLabels[task.status] ?? {
    label: task.status,
    className: "",
  };

  if (task.status === "failed" && task.last_error_summary) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className={`cursor-help text-sm underline decoration-dashed ${status.className}`}
            >
              {kind} {status.label}
            </span>
          </TooltipTrigger>
          <TooltipContent side="bottom" className="max-w-sm">
            <p className="text-xs break-all">{task.last_error_summary}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return (
    <span className={`text-sm ${status.className}`}>
      {kind} {status.label}
    </span>
  );
}
