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
    <div className="space-y-4">
      {bindings.length === 0 ? (
        <p className="text-sm text-muted-foreground">该主机未绑定出口 IP</p>
      ) : (
        <ul className="space-y-2">
          {bindings.map((b) => (
            <li
              key={b.binding_id}
              className="flex items-center justify-between rounded-md border px-3 py-2"
            >
              <div className="text-sm">
                <span className="font-medium">{b.egress_ip.label}</span>
                <span className="ml-2 font-mono text-muted-foreground">
                  {b.egress_ip.ip_address}
                </span>
              </div>
              {isRunning ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span>
                      <Button variant="ghost" size="icon" disabled>
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

      <div className="flex items-center gap-2">
        {isRunning ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="flex-1">
                <Select disabled>
                  <SelectTrigger>
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
          <Select value={selectedIpId} onValueChange={setSelectedIpId}>
            <SelectTrigger className="flex-1">
              <SelectValue placeholder="选择出口 IP" />
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
        )}
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Button
                size="icon"
                onClick={handleBind}
                disabled={isRunning || !selectedIpId || bindMutation.isPending}
              >
                <Plus className="h-4 w-4" />
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
  );
}
