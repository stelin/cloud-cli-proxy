import { createRootRoute, Outlet } from "@tanstack/react-router";
import { Toaster } from "sonner";
import { ErrorBoundary } from "@/components/layout/error-boundary";

export const Route = createRootRoute({
  notFoundComponent: () => (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 p-8">
      <div className="flex h-20 w-20 items-center justify-center rounded-2xl bg-muted">
        <span className="text-4xl font-bold text-muted-foreground">404</span>
      </div>
      <h1 className="text-2xl font-bold">页面未找到</h1>
      <p className="max-w-md text-center text-sm text-muted-foreground">
        您访问的页面不存在或已被移除。
      </p>
      <a
        href="/"
        className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
      >
        返回首页
      </a>
    </div>
  ),
  component: () => (
    <ErrorBoundary>
      <Outlet />
      <Toaster position="top-right" />
    </ErrorBoundary>
  ),
});
