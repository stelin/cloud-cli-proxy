import { useState } from "react";
import { Copy, Check, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { useRotateHostSSHPassword } from "@/hooks/use-hosts";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface RotateHostSSHPasswordDialogProps {
  hostId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RotateHostSSHPasswordDialog({
  hostId,
  open,
  onOpenChange,
}: RotateHostSSHPasswordDialogProps) {
  const rotate = useRotateHostSSHPassword();
  const [newPassword, setNewPassword] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  function handleRotate() {
    rotate.mutate(hostId, {
      onSuccess: (data) => setNewPassword(data.new_password),
      onError: () => toast.error("重置主机 SSH 密码失败"),
    });
  }

  function handleCopy() {
    if (!newPassword) return;
    navigator.clipboard.writeText(newPassword).then(() => {
      setCopied(true);
      toast.success("已复制到剪贴板");
      setTimeout(() => setCopied(false), 2000);
    });
  }

  function handleClose(v: boolean) {
    if (!v) {
      setNewPassword(null);
      setCopied(false);
    }
    onOpenChange(v);
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>重置主机 SSH 密码</DialogTitle>
          <DialogDescription>
            该密码用于通过 SSH 代理直连这台主机。新密码仅展示一次。
          </DialogDescription>
        </DialogHeader>

        {newPassword ? (
          <div className="space-y-4">
            <div className="flex items-center gap-2 rounded-md border bg-muted p-3">
              <code className="flex-1 break-all font-mono text-sm">
                {newPassword}
              </code>
              <Button variant="ghost" size="icon" onClick={handleCopy}>
                {copied ? (
                  <Check className="h-4 w-4 text-green-600" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
            <div className="flex items-start gap-2 rounded-md bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-950/50 dark:text-amber-200">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>关闭弹窗后将无法再次查看明文，请先复制保存。</span>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            点击下方按钮将为这台主机生成新的 SSH 密码，旧密码会立即失效。
          </p>
        )}

        <DialogFooter>
          {newPassword ? (
            <Button onClick={() => handleClose(false)}>关闭</Button>
          ) : (
            <>
              <Button variant="outline" onClick={() => handleClose(false)}>
                取消
              </Button>
              <Button onClick={handleRotate} disabled={rotate.isPending}>
                {rotate.isPending ? "处理中…" : "重置 SSH 密码"}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
