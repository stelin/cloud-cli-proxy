import { useState } from "react";
import { toast } from "sonner";
import { useDeleteUser, type User } from "@/hooks/use-users";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogCancel,
} from "@/components/ui/alert-dialog";

interface DeleteUserDialogProps {
  user: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}

export function DeleteUserDialog({
  user,
  open,
  onOpenChange,
  onDeleted,
}: DeleteUserDialogProps) {
  const deleteUser = useDeleteUser();
  const [confirmText, setConfirmText] = useState("");

  const canDelete = confirmText === user.username;

  function handleDelete() {
    deleteUser.mutate(user.id, {
      onSuccess: () => {
        toast.success("用户已删除");
        setConfirmText("");
        onOpenChange(false);
        onDeleted?.();
      },
      onError: () => toast.error("删除失败"),
    });
  }

  return (
    <AlertDialog
      open={open}
      onOpenChange={(v) => {
        if (!v) setConfirmText("");
        onOpenChange(v);
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>确认删除用户</AlertDialogTitle>
          <AlertDialogDescription>
            删除用户将同时清理其所有主机和绑定关系，此操作不可逆。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="space-y-2">
          <Label htmlFor="confirm-username">
            请输入用户名 <strong>{user.username}</strong> 以确认删除
          </Label>
          <Input
            id="confirm-username"
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            placeholder={user.username}
          />
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => setConfirmText("")}>
            取消
          </AlertDialogCancel>
          <Button
            variant="destructive"
            disabled={!canDelete || deleteUser.isPending}
            onClick={handleDelete}
          >
            {deleteUser.isPending ? "删除中…" : "确认删除"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
