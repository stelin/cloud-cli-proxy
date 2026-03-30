import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Copy, Check } from "lucide-react";
import { useCreateUser } from "@/hooks/use-users";
import { ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

const schema = z.object({
  username: z
    .string()
    .min(3, "用户名至少 3 个字符")
    .max(50, "用户名最多 50 个字符"),
});

type FormValues = z.infer<typeof schema>;

interface Credentials {
  password: string;
  short_id: string;
}

interface CreateUserDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function CopyField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  function copy() {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <div className="flex items-center gap-2">
        <code className="flex-1 rounded bg-muted px-3 py-2 text-sm font-mono">
          {value}
        </code>
        <Button type="button" variant="ghost" size="icon" onClick={copy}>
          {copied ? (
            <Check className="h-4 w-4 text-green-500" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </Button>
      </div>
    </div>
  );
}

export function CreateUserDialog({
  open,
  onOpenChange,
}: CreateUserDialogProps) {
  const createUser = useCreateUser();
  const [credentials, setCredentials] = useState<Credentials | null>(null);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
  });

  function handleClose() {
    reset();
    setCredentials(null);
    onOpenChange(false);
  }

  function onSubmit(data: FormValues) {
    createUser.mutate(data, {
      onSuccess: (res) => {
        toast.success("用户创建成功");
        setCredentials({
          password: res.password,
          short_id: res.short_id,
        });
      },
      onError: (err) => {
        if (err instanceof ApiError && err.status === 409) {
          toast.error("用户名已存在");
        } else {
          toast.error("创建失败");
        }
      },
    });
  }

  if (credentials) {
    return (
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>用户创建成功</DialogTitle>
            <DialogDescription>
              请妥善保存以下凭据，密码仅显示一次。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <CopyField label="登录密码" value={credentials.password} />
            <CopyField label="用户短 ID" value={credentials.short_id} />
            <p className="text-sm text-muted-foreground">
              SSH 短 ID 和 SSH 密码会在创建主机时按主机单独生成，不在创建用户时发放。
            </p>
          </div>
          <DialogFooter>
            <Button onClick={handleClose}>关闭</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>创建用户</DialogTitle>
          <DialogDescription>
              创建用户后系统会生成登录密码和用户短 ID。主机 SSH 凭据需在创建主机时单独生成。
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="username">用户名</Label>
            <Input
              id="username"
              placeholder="输入用户名"
              {...register("username")}
            />
            {errors.username && (
              <p className="text-sm text-destructive">
                {errors.username.message}
              </p>
            )}
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type="submit" disabled={createUser.isPending}>
              {createUser.isPending ? "创建中…" : "创建"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
