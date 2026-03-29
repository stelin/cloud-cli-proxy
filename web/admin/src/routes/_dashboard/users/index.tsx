import { useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { MoreHorizontal, Plus, Eye, Ban, CheckCircle, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useUsers, useUpdateUserStatus, type User } from "@/hooks/use-users";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { CreateUserDialog } from "@/components/users/create-user-dialog";
import { DeleteUserDialog } from "@/components/users/delete-user-dialog";

export const Route = createFileRoute("/_dashboard/users/")({
  component: UsersPage,
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

function UsersPage() {
  const { data, isLoading } = useUsers();
  const updateStatus = useUpdateUserStatus();
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteUser, setDeleteUser] = useState<User | null>(null);

  const users = data?.users ?? [];

  function handleToggleStatus(user: User) {
    const newStatus = user.status === "active" ? "disabled" : "active";
    const label =
      newStatus === "active"
        ? user.status === "expired"
          ? "用户已重新激活"
          : "用户已启用"
        : "用户已禁用";
    updateStatus.mutate(
      { userId: user.id, status: newStatus },
      {
        onSuccess: () => toast.success(label),
        onError: () => toast.error("操作失败"),
      },
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">用户管理</h1>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          创建用户
        </Button>
      </div>

      <div className="rounded-md border bg-background">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>用户名</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>到期时间</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead className="w-[60px]" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <div className="h-4 w-24 animate-pulse rounded bg-muted" />
                  </TableCell>
                  <TableCell>
                    <div className="h-5 w-14 animate-pulse rounded-full bg-muted" />
                  </TableCell>
                  <TableCell>
                    <div className="h-4 w-28 animate-pulse rounded bg-muted" />
                  </TableCell>
                  <TableCell>
                    <div className="h-4 w-32 animate-pulse rounded bg-muted" />
                  </TableCell>
                  <TableCell />
                </TableRow>
              ))
            ) : users.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={5}
                  className="h-24 text-center text-muted-foreground"
                >
                  暂无用户
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell>
                    <Link
                      to="/users/$userId"
                      params={{ userId: user.id }}
                      className="font-medium text-primary hover:underline"
                    >
                      {user.username}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        user.status === "active"
                          ? "default"
                          : user.status === "expired"
                            ? "destructive"
                            : "secondary"
                      }
                    >
                      {user.status === "active"
                        ? "活跃"
                        : user.status === "expired"
                          ? "已过期"
                          : "已禁用"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {user.expires_at ? formatDate(user.expires_at) : "永不过期"}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(user.created_at)}
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem asChild>
                          <Link
                            to="/users/$userId"
                            params={{ userId: user.id }}
                          >
                            <Eye />
                            查看详情
                          </Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => handleToggleStatus(user)}
                        >
                          {user.status === "active" ? (
                            <>
                              <Ban />
                              禁用用户
                            </>
                          ) : user.status === "expired" ? (
                            <>
                              <CheckCircle />
                              重新激活
                            </>
                          ) : (
                            <>
                              <CheckCircle />
                              启用用户
                            </>
                          )}
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() => setDeleteUser(user)}
                        >
                          <Trash2 />
                          删除用户
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <CreateUserDialog open={createOpen} onOpenChange={setCreateOpen} />
      {deleteUser && (
        <DeleteUserDialog
          user={deleteUser}
          open={!!deleteUser}
          onOpenChange={(open) => {
            if (!open) setDeleteUser(null);
          }}
        />
      )}
    </div>
  );
}
