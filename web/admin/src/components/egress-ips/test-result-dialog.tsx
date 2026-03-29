import { Check, X, AlertCircle, Minus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import type { TestResult } from "@/hooks/use-egress-ips";

interface TestResultDialogProps {
  result: TestResult | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function TestResultDialog({
  result,
  open,
  onOpenChange,
}: TestResultDialogProps) {
  if (!result) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            出口 IP 测试结果
            <StatusBadge status={result.status} />
          </DialogTitle>
        </DialogHeader>

        {result.message && (
          <p className="text-sm text-muted-foreground">{result.message}</p>
        )}

        {result.results && (
          <div className="space-y-4">
            <ConnectivitySection result={result.results.connectivity} />
            <EgressIPSection result={result.results.egress_ip} />
            <DNSLeakSection result={result.results.dns_leak} />
          </div>
        )}

        <p className="text-xs text-muted-foreground">
          测试时间：{formatDateTime(result.tested_at)}
        </p>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            关闭
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function StatusBadge({ status }: { status: string }) {
  switch (status) {
    case "passed":
      return (
        <Badge className="bg-green-100 text-green-700 hover:bg-green-100">
          全部通过
        </Badge>
      );
    case "partial":
      return (
        <Badge className="bg-yellow-100 text-yellow-700 hover:bg-yellow-100">
          部分通过
        </Badge>
      );
    case "failed":
      return (
        <Badge className="bg-red-100 text-red-700 hover:bg-red-100">
          全部失败
        </Badge>
      );
    default:
      return <Badge variant="secondary">测试错误</Badge>;
  }
}

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case "pass":
      return (
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100">
          <Check className="h-3 w-3 text-green-600" />
        </span>
      );
    case "fail":
      return (
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-red-100">
          <X className="h-3 w-3 text-red-600" />
        </span>
      );
    default:
      return (
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-gray-100">
          <AlertCircle className="h-3 w-3 text-gray-500" />
        </span>
      );
  }
}

function ConnectivitySection({
  result,
}: {
  result: TestResult["results"]["connectivity"];
}) {
  return (
    <div className="flex items-start gap-3 rounded-lg border p-3">
      <StatusIcon status={result.status} />
      <div className="flex-1">
        <p className="text-sm font-medium">连通性检测</p>
        {result.status === "pass" && result.latency_ms !== undefined && (
          <p className="text-xs text-muted-foreground">
            延迟: {result.latency_ms}ms
          </p>
        )}
        {result.error && <p className="text-xs text-red-600">{result.error}</p>}
      </div>
    </div>
  );
}

function EgressIPSection({
  result,
}: {
  result: TestResult["results"]["egress_ip"];
}) {
  return (
    <div className="flex items-start gap-3 rounded-lg border p-3">
      <StatusIcon status={result.status} />
      <div className="flex-1">
        <p className="text-sm font-medium">出口 IP 检测</p>
        {result.ip && (
          <p className="mt-1 text-sm">
            出口 IP: <span className="font-mono font-semibold">{result.ip}</span>
          </p>
        )}
        {result.sources && Object.keys(result.sources).length > 0 && (
          <div className="mt-2 space-y-0.5 text-xs text-muted-foreground">
            <p className="font-medium">检测来源:</p>
            {Object.entries(result.sources).map(([source, ip]) => (
              <p key={source} className="pl-2">
                {source}: <span className="font-mono">{ip}</span>
              </p>
            ))}
          </div>
        )}
        {result.error && <p className="text-xs text-red-600">{result.error}</p>}
      </div>
    </div>
  );
}

function DNSLeakSection({
  result,
}: {
  result: TestResult["results"]["dns_leak"];
}) {
  if (result.status === "skip") {
    return (
      <div className="flex items-start gap-3 rounded-lg border border-dashed p-3">
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-gray-100">
          <Minus className="h-3 w-3 text-gray-400" />
        </span>
        <div className="flex-1">
          <p className="text-sm font-medium text-muted-foreground">DNS 泄漏检测</p>
          <p className="text-xs text-muted-foreground">
            仅在容器运行时检测，探针测试不适用
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-start gap-3 rounded-lg border p-3">
      <StatusIcon status={result.status} />
      <div className="flex-1">
        <p className="text-sm font-medium">DNS 泄漏检测</p>
        {result.dns_servers_detected &&
          result.dns_servers_detected.length > 0 && (
            <div className="mt-1 text-xs text-muted-foreground">
              <p>检测到的 DNS 服务器:</p>
              {result.dns_servers_detected.map((dns) => (
                <p key={dns} className="pl-2 font-mono">
                  {dns}
                </p>
              ))}
            </div>
          )}
        {result.local_dns_leaked && (
          <p className="mt-1 text-xs text-red-600">检测到本地 DNS 泄漏</p>
        )}
        {result.error && <p className="text-xs text-red-600">{result.error}</p>}
      </div>
    </div>
  );
}

function formatDateTime(dateStr: string) {
  return new Date(dateStr).toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}
