import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Copy, Check, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { useRotateSSHPassword } from "@/hooks/use-users";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface RotateSSHPasswordDialogProps {
  userId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RotateSSHPasswordDialog({
  userId,
  open,
  onOpenChange,
}: RotateSSHPasswordDialogProps) {
  const qc = useQueryClient();
  const rotate = useRotateSSHPassword();
  const [newPassword, setNewPassword] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [useCustom, setUseCustom] = useState(false);
  const [customInput, setCustomInput] = useState("");

  function handleRotate() {
    if (useCustom) {
      const t = customInput.trim();
      if (t.length < 6) {
        toast.error("自定义 SSH 密码至少 6 个字符");
        return;
      }
    }
    rotate.mutate(
      {
        userId,
        newPassword: useCustom ? customInput.trim() : undefined,
      },
      {
        onSuccess: (data) => {
          setNewPassword(data.new_password);
          void qc.invalidateQueries({ queryKey: ["users", userId] });
        },
        onError: () => toast.error("重置 SSH 密码失败"),
      },
    );
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
      setUseCustom(false);
      setCustomInput("");
    }
    onOpenChange(v);
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>重置 SSH 密码</DialogTitle>
          <DialogDescription>
            用于 SSH 登录容器（与网页/登录密码不同）。可随机生成或指定新密码。
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
          <div className="space-y-4">
            <label className="flex cursor-pointer items-center gap-2 text-sm">
              <input
                id="ssh-custom"
                type="checkbox"
                checked={useCustom}
                onChange={(e) => setUseCustom(e.target.checked)}
                className="h-4 w-4 rounded border border-input"
              />
              <span>指定新 SSH 密码（否则随机生成）</span>
            </label>
            {useCustom && (
              <div className="space-y-2">
                <Label htmlFor="ssh-pw">新 SSH 密码</Label>
                <Input
                  id="ssh-pw"
                  type="password"
                  autoComplete="new-password"
                  value={customInput}
                  onChange={(e) => setCustomInput(e.target.value)}
                  placeholder="6–128 个字符"
                />
              </div>
            )}
            {!useCustom && (
              <p className="text-sm text-muted-foreground">
                点击下方按钮将生成随机 SSH 密码。
              </p>
            )}
          </div>
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
