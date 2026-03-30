import { useState } from "react";
import { X, Plus } from "lucide-react";
import { toast } from "sonner";
import {
  useBindEgressIP,
  useUnbindEgressIP,
  type HostBinding,
} from "@/hooks/use-hosts";
import { useEgressIPs } from "@/hooks/use-egress-ips";
import { ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface BindingManagerProps {
  hostId: string;
  hostStatus: string;
  bindings: HostBinding[];
}

export function BindingManager({
  hostId,
  hostStatus,
  bindings,
}: BindingManagerProps) {
  const isRunning = hostStatus === "running";
  const bindMutation = useBindEgressIP();
  const unbindMutation = useUnbindEgressIP();
  const { data: egressData } = useEgressIPs();
  const [selectedIpId, setSelectedIpId] = useState<string>("");

  const availableIPs =
    egressData?.egress_ips?.filter((ip) => ip.status === "available") ?? [];

  const boundIpIds = new Set(bindings.map((b) => b.egress_ip.id));
  const unboundIPs = availableIPs.filter((ip) => !boundIpIds.has(ip.id));

  function handleBind() {
    if (!selectedIpId) return;
    bindMutation.mutate(
      { host_id: hostId, egress_ip_id: selectedIpId },
      {
        onSuccess: () => {
          toast.success("绑定成功");
          setSelectedIpId("");
        },
        onError: (err) => {
          if (err instanceof ApiError && err.status === 409) {
            toast.error("运行中主机不允许绑定出口 IP，请先停止主机");
          } else {
            toast.error("绑定失败");
          }
        },
      },
    );
  }

  function handleUnbind(bindingId: string) {
    unbindMutation.mutate(bindingId, {
      onSuccess: () => toast.success("已解绑"),
      onError: (err) => {
        if (err instanceof ApiError && err.status === 409) {
          toast.error("运行中主机不允许解绑出口 IP，请先停止主机");
        } else {
          toast.error("解绑失败");
        }
      },
    });
  }

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          当前绑定
        </p>
        {bindings.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border/80 bg-muted/20 px-4 py-8 text-center">
            <p className="text-sm text-muted-foreground">尚未绑定出口 IP</p>
            <p className="mt-1 text-xs text-muted-foreground">
              在下方选择出口并点击添加，绑定后流量才会走指定出口。
            </p>
          </div>
        ) : (
          <ul className="space-y-3">
            {bindings.map((b) => (
              <li
                key={b.binding_id}
                className="flex items-center justify-between gap-3 rounded-xl border border-border/80 bg-card px-4 py-3.5 shadow-sm"
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium leading-tight">
                    {b.egress_ip.label}
                  </p>
                  <p className="mt-1 font-mono text-xs text-muted-foreground">
                    {b.egress_ip.ip_address}
                  </p>
                </div>
                {isRunning ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span>
                        <Button variant="ghost" size="icon" className="shrink-0" disabled>
                          <X className="h-4 w-4" />
                        </Button>
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      运行中主机不允许解绑出口 IP，请先停止主机
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="shrink-0"
                    onClick={() => handleUnbind(b.binding_id)}
                    disabled={unbindMutation.isPending}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="rounded-xl border border-border/60 bg-muted/20 p-4">
        <Label htmlFor="egress-bind-select" className="text-xs font-semibold text-foreground">
          添加绑定
        </Label>
        <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end">
          {isRunning ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="w-full flex-1">
                  <Select disabled>
                    <SelectTrigger id="egress-bind-select" className="h-11 w-full">
                      <SelectValue placeholder="选择出口 IP" />
                    </SelectTrigger>
                  </Select>
                </span>
              </TooltipTrigger>
              <TooltipContent>
                运行中主机不允许绑定出口 IP，请先停止主机
              </TooltipContent>
            </Tooltip>
          ) : (
            <div className="w-full flex-1 space-y-2">
              <Select value={selectedIpId} onValueChange={setSelectedIpId}>
                <SelectTrigger id="egress-bind-select" className="h-11 w-full">
                  <SelectValue placeholder="选择要绑定的出口 IP" />
                </SelectTrigger>
                <SelectContent>
                  {unboundIPs.map((ip) => (
                    <SelectItem key={ip.id} value={ip.id}>
                      {ip.label} ({ip.ip_address})
                    </SelectItem>
                  ))}
                  {unboundIPs.length === 0 && (
                    <SelectItem value="_none" disabled>
                      无可用出口 IP
                    </SelectItem>
                  )}
                </SelectContent>
              </Select>
            </div>
          )}
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="sm:w-auto">
                <Button
                  type="button"
                  className="h-11 w-full gap-2 sm:w-auto sm:min-w-28"
                  onClick={handleBind}
                  disabled={isRunning || !selectedIpId || bindMutation.isPending}
                >
                  <Plus className="h-4 w-4" />
                  添加绑定
                </Button>
              </span>
            </TooltipTrigger>
            {isRunning && (
              <TooltipContent>
                运行中主机不允许绑定出口 IP，请先停止主机
              </TooltipContent>
            )}
          </Tooltip>
        </div>
      </div>
    </div>
  );
}
