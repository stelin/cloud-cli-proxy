import { useState } from "react";
import { Copy, Check, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { useRotatePassword } from "@/hooks/use-users";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

interface RotatePasswordDialogProps {
  userId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RotatePasswordDialog({
  userId,
  open,
  onOpenChange,
}: RotatePasswordDialogProps) {
  const rotate = useRotatePassword();
  const [newPassword, setNewPassword] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  function handleRotate() {
    rotate.mutate(userId, {
      onSuccess: (data) => {
        setNewPassword(data.new_password);
      },
      onError: () => toast.error("密码轮换失败"),
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
          <DialogTitle>轮换密码</DialogTitle>
          <DialogDescription>
            系统将自动生成一个新的随机强密码。
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
              <span>新密码仅展示一次，关闭后无法找回。</span>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            点击下方按钮生成新密码，旧密码将立即失效。
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
                {rotate.isPending ? "生成中…" : "轮换密码"}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
