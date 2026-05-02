import { useState } from "react";
import { X, Plus } from "lucide-react";
import { toast } from "sonner";
import { useUpdateHostPorts, type HostPort } from "@/hooks/use-hosts";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

interface PortManagerProps {
  hostId: string;
  hostStatus: string;
  ports: HostPort[];
}

export function PortManager({ hostId, hostStatus, ports }: PortManagerProps) {
  const isRunning = hostStatus === "running";
  const updatePortsMutation = useUpdateHostPorts(hostId);

  const [localPorts, setLocalPorts] = useState<HostPort[]>(ports);
  const [newHostPort, setNewHostPort] = useState("");
  const [newContainerPort, setNewContainerPort] = useState("");
  const [newProtocol, setNewProtocol] = useState("tcp");

  function handleAdd() {
    const hp = parseInt(newHostPort, 10);
    const cp = parseInt(newContainerPort, 10);
    if (isNaN(hp) || hp <= 0 || hp > 65535 || isNaN(cp) || cp <= 0 || cp > 65535) {
      return;
    }
    setLocalPorts([
      ...localPorts,
      { host_port: hp, container_port: cp, protocol: newProtocol },
    ]);
    setNewHostPort("");
    setNewContainerPort("");
    setNewProtocol("tcp");
  }

  function handleRemove(index: number) {
    setLocalPorts(localPorts.filter((_, i) => i !== index));
  }

  function handleSave() {
    updatePortsMutation.mutate(localPorts, {
      onSuccess: () => toast.success("端口映射配置已保存"),
      onError: () => toast.error("保存失败"),
    });
  }

  const hasChanges =
    JSON.stringify(localPorts) !== JSON.stringify(ports);

  const hostPortNum = parseInt(newHostPort, 10);
  const containerPortNum = parseInt(newContainerPort, 10);
  const addDisabled =
    isNaN(hostPortNum) ||
    hostPortNum <= 0 ||
    hostPortNum > 65535 ||
    isNaN(containerPortNum) ||
    containerPortNum <= 0 ||
    containerPortNum > 65535;

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          已配置端口映射
        </p>
        {localPorts.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border/80 bg-muted/20 px-4 py-8 text-center">
            <p className="text-sm text-muted-foreground">暂无端口映射</p>
            <p className="mt-1 text-xs text-muted-foreground">
              在下方填写宿主机端口与容器端口，添加后保存即可生效。
            </p>
          </div>
        ) : (
          <ul className="space-y-3">
            {localPorts.map((p, i) => (
              <li
                key={i}
                className="flex items-center justify-between gap-3 rounded-xl border border-border/80 bg-card px-4 py-3.5 shadow-sm"
              >
                <div className="min-w-0 flex-1">
                  <p className="font-mono text-sm leading-tight">
                    {p.host_port}:{p.container_port}
                    {p.protocol && p.protocol !== "tcp" && (
                      <span className="text-muted-foreground">/{p.protocol}</span>
                    )}
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    宿主机 {p.host_port} → 容器 {p.container_port}
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
                      运行中主机不允许修改端口映射，请先停止主机
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="shrink-0"
                    onClick={() => handleRemove(i)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>

      {!isRunning && (
        <div className="rounded-xl border border-border/60 bg-muted/20 p-4">
          <Label className="text-xs font-semibold text-foreground">
            添加端口映射
          </Label>
          <div className="mt-3 space-y-3">
            <div className="grid gap-3 sm:grid-cols-[1fr_auto_1fr_auto] sm:items-end">
              <div className="space-y-1">
                <span className="text-xs text-muted-foreground">宿主机端口</span>
                <Input
                  type="number"
                  min={1}
                  max={65535}
                  placeholder="如 8080"
                  value={newHostPort}
                  onChange={(e) => setNewHostPort(e.target.value)}
                />
              </div>
              <span className="hidden pb-2.5 text-muted-foreground sm:block">:</span>
              <div className="space-y-1">
                <span className="text-xs text-muted-foreground">容器端口</span>
                <Input
                  type="number"
                  min={1}
                  max={65535}
                  placeholder="如 80"
                  value={newContainerPort}
                  onChange={(e) => setNewContainerPort(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <span className="text-xs text-muted-foreground">协议</span>
                <Select value={newProtocol} onValueChange={setNewProtocol}>
                  <SelectTrigger className="w-[80px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="tcp">TCP</SelectItem>
                    <SelectItem value="udp">UDP</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            {newHostPort !== "" && (hostPortNum <= 0 || hostPortNum > 65535) && (
              <p className="text-xs text-destructive">宿主机端口必须在 1~65535 之间</p>
            )}
            {newContainerPort !== "" && (containerPortNum <= 0 || containerPortNum > 65535) && (
              <p className="text-xs text-destructive">容器端口必须在 1~65535 之间</p>
            )}
            <Button
              type="button"
              variant="outline"
              className="gap-2"
              disabled={addDisabled}
              onClick={handleAdd}
            >
              <Plus className="h-4 w-4" />
              添加
            </Button>
          </div>
        </div>
      )}

      {!isRunning && hasChanges && (
        <Button
          type="button"
          onClick={handleSave}
          disabled={updatePortsMutation.isPending}
        >
          {updatePortsMutation.isPending ? "保存中..." : "保存端口映射"}
        </Button>
      )}
    </div>
  );
}
