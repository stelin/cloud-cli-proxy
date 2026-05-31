import { useState } from "react";
import { Folder, File, ChevronRight, Home, FolderOpen } from "lucide-react";
import { useHostFiles } from "@/hooks/use-host-files";
import { DataTableShell } from "@/components/layout/data-table-shell";
import { EmptyState } from "@/components/layout/empty-state";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }
  if (bytes >= 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${bytes} B`;
}

function formatModTime(dateStr: string): string {
  if (!dateStr) return "—";
  try {
    return new Date(dateStr).toLocaleDateString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return "—";
  }
}

function buildBreadcrumbs(
  currentPath: string,
): { label: string; path: string }[] {
  if (currentPath === "/") return [{ label: "根目录", path: "/" }];

  const parts = currentPath.split("/").filter(Boolean);
  const breadcrumbs = [{ label: "根目录", path: "/" }];
  let accumulated = "";
  for (const part of parts) {
    accumulated += `/${part}`;
    breadcrumbs.push({ label: part, path: accumulated });
  }
  return breadcrumbs;
}

interface HostFilesBrowserProps {
  hostId: string;
}

export function HostFilesBrowser({ hostId }: HostFilesBrowserProps) {
  const [currentPath, setCurrentPath] = useState("/");

  const { data, isLoading, isError, error, refetch } = useHostFiles(
    currentPath,
    hostId,
  );
  const entries = data?.entries ?? [];
  const breadcrumbs = buildBreadcrumbs(currentPath);

  return (
    <div className="space-y-4">
      {/* 面包屑导航 */}
      <nav
        className="flex flex-wrap items-center gap-1 text-sm"
        aria-label="面包屑导航"
      >
        {breadcrumbs.map((crumb, index) => (
          <span key={crumb.path} className="flex items-center gap-1">
            {index > 0 && (
              <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            )}
            {index === 0 ? (
              <button
                type="button"
                onClick={() => setCurrentPath(crumb.path)}
                className="flex items-center gap-1 rounded px-1.5 py-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              >
                <Home className="h-3.5 w-3.5" />
                {crumb.label}
              </button>
            ) : index === breadcrumbs.length - 1 ? (
              <span className="flex items-center gap-1 rounded px-1.5 py-0.5 font-medium text-foreground">
                <FolderOpen className="h-3.5 w-3.5" />
                {crumb.label}
              </span>
            ) : (
              <button
                type="button"
                onClick={() => setCurrentPath(crumb.path)}
                className="rounded px-1.5 py-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              >
                {crumb.label}
              </button>
            )}
          </span>
        ))}
      </nav>

      <DataTableShell>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead className="w-[120px]">大小</TableHead>
              <TableHead className="w-[180px]">修改时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 8 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 3 }).map((_, j) => (
                    <TableCell key={j}>
                      <div className="h-4 w-20 animate-pulse rounded bg-muted" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : isError ? (
              <TableRow>
                <TableCell colSpan={3} className="p-0">
                  <EmptyState
                    icon={Folder}
                    title="加载失败"
                    description={
                      error instanceof Error
                        ? error.message
                        : "无法加载文件列表，请稍后重试"
                    }
                    action={
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => refetch()}
                      >
                        重试
                      </Button>
                    }
                  />
                </TableCell>
              </TableRow>
            ) : entries.length === 0 ? (
              <TableRow>
                <TableCell colSpan={3} className="p-0">
                  <EmptyState
                    icon={FolderOpen}
                    title="空目录"
                    description="当前目录下没有任何文件或子目录"
                  />
                </TableCell>
              </TableRow>
            ) : (
              entries.map((entry) => (
                <TableRow key={entry.path}>
                  <TableCell>
                    {entry.is_dir ? (
                      <button
                        type="button"
                        onClick={() => setCurrentPath(entry.path)}
                        className="flex items-center gap-2 text-left transition-colors hover:text-primary"
                      >
                        <Folder className="h-4 w-4 shrink-0 text-blue-500" />
                        <span className="font-medium">{entry.name}</span>
                      </button>
                    ) : (
                      <span className="flex items-center gap-2">
                        <File className="h-4 w-4 shrink-0 text-muted-foreground" />
                        <span>{entry.name}</span>
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {entry.is_dir ? "—" : formatFileSize(entry.size)}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {formatModTime(entry.mod_time)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </DataTableShell>
    </div>
  );
}
