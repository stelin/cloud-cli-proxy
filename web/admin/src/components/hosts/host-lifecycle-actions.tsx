import { useState } from "react";
import { Play, Square, RefreshCw, Trash2, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { useNavigate } from "@tanstack/react-router";
import { useHostAction, useDeleteHost } from "@/hooks/use-hosts";
import { Button } from "@/components/ui/button";
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
import { RebuildDialog } from "./rebuild-dialog";

interface HostLifecycleActionsProps {
  hostId: string;
  hostStatus: string;
}

export function HostLifecycleActions({
  hostId,
  hostStatus,
}: HostLifecycleActionsProps) {
  const navigate = useNavigate();
  const actionMutation = useHostAction();
  const deleteMutation = useDeleteHost();
  const [rebuildOpen, setRebuildOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [forceDeleteOpen, setForceDeleteOpen] = useState(false);

  function handleAction(action: "start" | "stop") {
    actionMutation.mutate(
      { hostId, action },
      {
        onSuccess: () => toast.success("操作已提交，请查看任务状态"),
        onError: () => toast.error("操作提交失败"),
      },
    );
  }

  function handleDelete(force: boolean) {
    deleteMutation.mutate(
      { hostId, force },
      {
        onSuccess: () => {
          toast.success("主机已删除");
          navigate({ to: "/hosts" });
        },
        onError: (err: any) => {
          const msg = err?.message || "删除失败";
          if (msg.includes("运行中")) {
            toast.error("主机正在运行中，请先停止或使用强制删除");
          } else {
            toast.error(msg);
          }
        },
      },
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-3">
        {(hostStatus === "stopped" || hostStatus === "failed") && (
          <Button
            onClick={() => handleAction("start")}
            disabled={actionMutation.isPending}
          >
            <Play className="h-4 w-4" />
            启动
          </Button>
        )}

        {hostStatus === "running" && (
          <Button
            variant="secondary"
            onClick={() => handleAction("stop")}
            disabled={actionMutation.isPending}
          >
            <Square className="h-4 w-4" />
            停止
          </Button>
        )}

        <Button
          variant="outline"
          onClick={() => setRebuildOpen(true)}
          disabled={actionMutation.isPending}
        >
          <RefreshCw className="h-4 w-4" />
          重建
        </Button>
      </div>

      <div className="flex flex-wrap gap-3 border-t pt-4">
        {hostStatus !== "running" ? (
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setDeleteOpen(true)}
            disabled={deleteMutation.isPending}
          >
            <Trash2 className="h-4 w-4" />
            删除主机
          </Button>
        ) : (
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setForceDeleteOpen(true)}
            disabled={deleteMutation.isPending}
          >
            <AlertTriangle className="h-4 w-4" />
            强制删除
          </Button>
        )}
      </div>

      <RebuildDialog
        hostId={hostId}
        open={rebuildOpen}
        onOpenChange={setRebuildOpen}
      />

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除主机？</AlertDialogTitle>
            <AlertDialogDescription>
              将停止并移除 Docker 容器，删除数据库中的主机记录和出口 IP
              绑定。此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => handleDelete(false)}
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={forceDeleteOpen} onOpenChange={setForceDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              <span className="flex items-center gap-2 text-destructive">
                <AlertTriangle className="h-5 w-5" />
                强制删除运行中的主机？
              </span>
            </AlertDialogTitle>
            <AlertDialogDescription>
              主机当前正在运行，强制删除将立即终止容器并清除所有数据。此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => handleDelete(true)}
            >
              强制删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
