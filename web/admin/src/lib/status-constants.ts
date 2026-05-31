// 统一的宿主状态展示配置。Admin 和 Portal 共用此定义，
// 避免在多处硬编码导致不一致。

export interface HostStatusDisplay {
  label: string;
  dot: string;
  color: string;
  bg: string;
  border: string;
  badgeVariant: "default" | "secondary" | "destructive" | "outline";
}

export const hostStatusConfig: Record<string, HostStatusDisplay> = {
  running: {
    label: "运行中",
    dot: "bg-emerald-500",
    color: "text-emerald-700",
    bg: "bg-emerald-50",
    border: "border-emerald-200",
    badgeVariant: "default",
  },
  stopped: {
    label: "已停止",
    dot: "bg-slate-400",
    color: "text-slate-600",
    bg: "bg-slate-50",
    border: "border-slate-200",
    badgeVariant: "secondary",
  },
  pending: {
    label: "等待中",
    dot: "bg-amber-500",
    color: "text-amber-700",
    bg: "bg-amber-50",
    border: "border-amber-200",
    badgeVariant: "outline",
  },
  failed: {
    label: "失败",
    dot: "bg-red-500",
    color: "text-red-700",
    bg: "bg-red-50",
    border: "border-red-200",
    badgeVariant: "destructive",
  },
  rebuilding: {
    label: "重建中",
    dot: "bg-amber-500",
    color: "text-amber-700",
    bg: "bg-amber-50",
    border: "border-amber-200",
    badgeVariant: "outline",
  },
};

export interface TaskStatusDisplay {
  label: string;
  variant: "default" | "secondary" | "destructive" | "outline";
}

export const taskStatusConfig: Record<string, TaskStatusDisplay> = {
  pending: { label: "等待中", variant: "outline" },
  running: { label: "运行中", variant: "default" },
  succeeded: { label: "成功", variant: "default" },
  failed: { label: "失败", variant: "destructive" },
  canceled: { label: "已取消", variant: "secondary" },
};

export interface UserStatusDisplay {
  label: string;
  dotColor: string;
  badgeVariant: "default" | "secondary" | "destructive" | "outline";
}

export const userStatusConfig: Record<string, UserStatusDisplay> = {
  active: { label: "活跃", dotColor: "bg-emerald-500", badgeVariant: "default" },
  disabled: { label: "已禁用", dotColor: "bg-slate-400", badgeVariant: "secondary" },
  expired: { label: "已过期", dotColor: "bg-red-500", badgeVariant: "destructive" },
};

export const defaultHostStatus: HostStatusDisplay = {
  label: "未知",
  dot: "bg-slate-400",
  color: "text-slate-600",
  bg: "bg-slate-50",
  border: "border-slate-200",
  badgeVariant: "outline",
};

export const defaultTaskStatus: TaskStatusDisplay = {
  label: "未知",
  variant: "outline",
};

export const defaultUserStatus: UserStatusDisplay = {
  label: "未知",
  dotColor: "bg-slate-400",
  badgeVariant: "outline",
};

export const taskKindLabels: Record<string, string> = {
  create_host: "创建",
  start_host: "启动",
  stop_host: "停止",
  rebuild_host: "重建",
  reload_host_bypass: "重载旁路",
  prepare_host: "准备",
  volume_remove: "移除卷",
};
