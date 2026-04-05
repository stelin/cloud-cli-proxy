import { useState, useMemo } from "react";
import {
  Copy,
  Check,
  Key,
  Download,
  Upload,
  Trash2,
  Plus,
  Shield,
  Globe,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import type { SSHKey } from "@/hooks/use-ssh-keys";

interface SSHKeyManagerProps {
  keys: SSHKey[];
  isLoading: boolean;
  onCreate: (params: {
    purpose: "inbound" | "outbound";
    label: string;
    keyType?: "ed25519" | "rsa";
    publicKey?: string;
    privateKey?: string;
  }) => void;
  onDelete: (keyId: string) => void;
  isCreating: boolean;
  isDeleting: boolean;
  lastCreatedKey?: SSHKey;
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function CopyButton({ text, label }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      toast.success(label || "已复制到剪贴板");
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <Button
      variant="ghost"
      size="sm"
      className="h-7 gap-1 text-xs"
      onClick={handleCopy}
    >
      {copied ? (
        <Check className="h-3.5 w-3.5 text-green-600" />
      ) : (
        <Copy className="h-3.5 w-3.5" />
      )}
      {copied ? "已复制" : "复制"}
    </Button>
  );
}

function KeyItemCard({
  sshKey,
  onDelete,
  isDeleting,
  showPublicKey,
}: {
  sshKey: SSHKey;
  onDelete: (keyId: string) => void;
  isDeleting: boolean;
  showPublicKey?: boolean;
}) {
  const [expanded, setExpanded] = useState(false);
  const isContainerOnly = sshKey.source === "container";

  return (
    <div className="rounded-lg border p-4 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <Key className="h-4 w-4 shrink-0 text-muted-foreground" />
          <span className="font-medium text-sm truncate">{sshKey.label}</span>
          {sshKey.key_type && (
            <Badge variant="outline" className="text-[10px] shrink-0">
              {sshKey.key_type}
            </Badge>
          )}
          {isContainerOnly && (
            <Badge variant="secondary" className="text-[10px] shrink-0">
              容器内
            </Badge>
          )}
          {sshKey.source === "managed" && sshKey.synced === false && (
            <Badge variant="destructive" className="text-[10px] shrink-0">
              未同步
            </Badge>
          )}
          {sshKey.source === "managed" && sshKey.synced === true && sshKey.purpose === "inbound" && (
            <Badge variant="outline" className="text-[10px] shrink-0 text-emerald-600 border-emerald-300">
              已同步
            </Badge>
          )}
        </div>
        {!isContainerOnly && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 gap-1 text-xs text-destructive hover:text-destructive shrink-0"
                disabled={isDeleting}
              >
                <Trash2 className="h-3.5 w-3.5" />
                删除
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>确认删除密钥「{sshKey.label}」？</AlertDialogTitle>
                <AlertDialogDescription>
                  删除后无法恢复。如果已在外部平台配置了该公钥，请一并清理。
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction
                  disabled={isDeleting}
                  onClick={() => onDelete(sshKey.id)}
                >
                  {isDeleting ? "删除中..." : "确认删除"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {sshKey.fingerprint && (
          <span className="font-mono">{sshKey.fingerprint}</span>
        )}
        {sshKey.created_at && <span>{formatDate(sshKey.created_at)}</span>}
      </div>

      {showPublicKey && sshKey.public_key && (
        <div className="space-y-1">
          <div className="flex items-center justify-between">
            <button
              type="button"
              className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              onClick={() => setExpanded(!expanded)}
            >
              {expanded ? "收起公钥" : "展开公钥"}
            </button>
            <CopyButton text={sshKey.public_key} label="公钥已复制" />
          </div>
          {expanded && (
            <pre className="max-h-24 overflow-auto whitespace-pre-wrap break-all rounded-lg border bg-muted/50 p-3 font-mono text-xs">
              {sshKey.public_key}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}

export function SSHKeyManager({
  keys,
  isLoading,
  onCreate,
  onDelete,
  isCreating,
  isDeleting,
  lastCreatedKey,
}: SSHKeyManagerProps) {
  const [inboundDialogOpen, setInboundDialogOpen] = useState(false);
  const [inboundLabel, setInboundLabel] = useState("");
  const [inboundPubKey, setInboundPubKey] = useState("");

  const [generateDialogOpen, setGenerateDialogOpen] = useState(false);
  const [generateLabel, setGenerateLabel] = useState("");
  const [generateKeyType, setGenerateKeyType] = useState<"ed25519" | "rsa">(
    "ed25519",
  );

  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [importLabel, setImportLabel] = useState("");
  const [importPubKey, setImportPubKey] = useState("");
  const [importPrivKey, setImportPrivKey] = useState("");

  const [createdKeyDialogOpen, setCreatedKeyDialogOpen] = useState(false);
  const [displayedCreatedKey, setDisplayedCreatedKey] = useState<SSHKey | null>(
    null,
  );

  const inboundKeys = useMemo(
    () =>
      keys
        .filter((k) => k.purpose === "inbound")
        .sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        ),
    [keys],
  );

  const outboundKeys = useMemo(
    () =>
      keys
        .filter((k) => k.purpose === "outbound")
        .sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        ),
    [keys],
  );

  function handleInboundSubmit() {
    const pub = inboundPubKey.trim();
    const label = inboundLabel.trim();
    if (!pub) {
      toast.error("公钥不能为空");
      return;
    }
    if (!label) {
      toast.error("标签不能为空");
      return;
    }
    onCreate({ purpose: "inbound", label, publicKey: pub });
    setInboundDialogOpen(false);
    setInboundLabel("");
    setInboundPubKey("");
  }

  function handleGenerateSubmit() {
    const label = generateLabel.trim();
    if (!label) {
      toast.error("标签不能为空");
      return;
    }
    onCreate({
      purpose: "outbound",
      label,
      keyType: generateKeyType,
    });
    setGenerateDialogOpen(false);
    setGenerateLabel("");
    setGenerateKeyType("ed25519");
  }

  function handleImportSubmit() {
    const label = importLabel.trim();
    const pub = importPubKey.trim();
    const priv = importPrivKey.trim();
    if (!label) {
      toast.error("标签不能为空");
      return;
    }
    if (!pub) {
      toast.error("公钥不能为空");
      return;
    }
    onCreate({
      purpose: "outbound",
      label,
      publicKey: pub,
      privateKey: priv || undefined,
    });
    setImportDialogOpen(false);
    setImportLabel("");
    setImportPubKey("");
    setImportPrivKey("");
  }

  function handleDownloadPrivKey(key: SSHKey) {
    if (!key.private_key) return;
    const blob = new Blob([key.private_key], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = key.key_type === "rsa" ? "id_rsa" : "id_ed25519";
    a.click();
    URL.revokeObjectURL(url);
    toast.success("私钥文件已下载");
  }

  const prevLastCreatedKeyId = useState<string | undefined>(undefined);
  if (
    lastCreatedKey &&
    lastCreatedKey.id !== prevLastCreatedKeyId[0] &&
    lastCreatedKey.purpose === "outbound" &&
    lastCreatedKey.private_key
  ) {
    prevLastCreatedKeyId[1](lastCreatedKey.id);
    setDisplayedCreatedKey(lastCreatedKey);
    setCreatedKeyDialogOpen(true);
  }

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Key className="h-5 w-5" />
            SSH 密钥管理
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-20 animate-pulse rounded bg-muted" />
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Key className="h-5 w-5" />
            SSH 密钥管理
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Inbound Keys Section */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <Shield className="h-4 w-4 text-muted-foreground" />
                  <h3 className="text-sm font-semibold">
                    入站密钥（免密登录）
                  </h3>
                </div>
                <p className="text-xs text-muted-foreground">
                  添加你本地电脑的 SSH 公钥，即可免密 SSH 连接到云主机。
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5 shrink-0"
                onClick={() => setInboundDialogOpen(true)}
                disabled={isCreating}
              >
                <Plus className="h-3.5 w-3.5" />
                添加公钥
              </Button>
            </div>

            {inboundKeys.length === 0 ? (
              <p className="text-sm text-muted-foreground py-3 text-center border rounded-lg bg-muted/20">
                暂无入站密钥
              </p>
            ) : (
              <div className="space-y-2">
                {inboundKeys.map((k) => (
                  <KeyItemCard
                    key={k.id}
                    sshKey={k}
                    onDelete={onDelete}
                    isDeleting={isDeleting}
                  />
                ))}
              </div>
            )}
          </div>

          <Separator />

          {/* Outbound Keys Section */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <Globe className="h-4 w-4 text-muted-foreground" />
                  <h3 className="text-sm font-semibold">
                    出站密钥（外部服务鉴权）
                  </h3>
                </div>
                <p className="text-xs text-muted-foreground">
                  生成密钥对并将公钥添加到 GitHub、GitLab
                  等平台，即可在云主机中 git clone 私有仓库。
                </p>
              </div>
              <div className="flex gap-2 shrink-0">
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-1.5"
                  onClick={() => setGenerateDialogOpen(true)}
                  disabled={isCreating}
                >
                  <Key className="h-3.5 w-3.5" />
                  生成密钥对
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-1.5"
                  onClick={() => setImportDialogOpen(true)}
                  disabled={isCreating}
                >
                  <Upload className="h-3.5 w-3.5" />
                  导入已有密钥
                </Button>
              </div>
            </div>

            {outboundKeys.length === 0 ? (
              <p className="text-sm text-muted-foreground py-3 text-center border rounded-lg bg-muted/20">
                暂无出站密钥
              </p>
            ) : (
              <div className="space-y-2">
                {outboundKeys.map((k) => (
                  <KeyItemCard
                    key={k.id}
                    sshKey={k}
                    onDelete={onDelete}
                    isDeleting={isDeleting}
                    showPublicKey
                  />
                ))}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Inbound: Add Public Key Dialog */}
      <Dialog open={inboundDialogOpen} onOpenChange={setInboundDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>添加入站公钥</DialogTitle>
            <DialogDescription>
              粘贴你本地电脑的 SSH 公钥，添加后即可免密连接到云主机。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="inbound-label">标签</Label>
              <Input
                id="inbound-label"
                placeholder="例如：MacBook Pro"
                value={inboundLabel}
                onChange={(e) => setInboundLabel(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="inbound-pubkey">公钥</Label>
              <textarea
                id="inbound-pubkey"
                className="w-full rounded-md border bg-muted/50 p-2 font-mono text-xs"
                rows={3}
                placeholder="ssh-ed25519 AAAA... 或 ssh-rsa AAAA..."
                value={inboundPubKey}
                onChange={(e) => setInboundPubKey(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setInboundDialogOpen(false)}
            >
              取消
            </Button>
            <Button
              onClick={handleInboundSubmit}
              disabled={!inboundPubKey.trim() || !inboundLabel.trim() || isCreating}
            >
              {isCreating ? "添加中..." : "添加"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Outbound: Generate Key Pair Dialog */}
      <Dialog open={generateDialogOpen} onOpenChange={setGenerateDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>生成出站密钥对</DialogTitle>
            <DialogDescription>
              后端将自动生成密钥对。生成后请立即复制公钥并下载私钥，私钥只在创建时可见。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="generate-label">标签</Label>
              <Input
                id="generate-label"
                placeholder="例如：GitHub / GitLab"
                value={generateLabel}
                onChange={(e) => setGenerateLabel(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>密钥类型</Label>
              <Select
                value={generateKeyType}
                onValueChange={(v) =>
                  setGenerateKeyType(v as "ed25519" | "rsa")
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ed25519">Ed25519（推荐）</SelectItem>
                  <SelectItem value="rsa">RSA 4096</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setGenerateDialogOpen(false)}
            >
              取消
            </Button>
            <Button
              onClick={handleGenerateSubmit}
              disabled={!generateLabel.trim() || isCreating}
            >
              {isCreating ? "生成中..." : "生成"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Outbound: Import Existing Key Dialog */}
      <Dialog open={importDialogOpen} onOpenChange={setImportDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>导入出站密钥</DialogTitle>
            <DialogDescription>
              粘贴你已有的 SSH 密钥对。公钥为必填项，私钥可选。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="import-label">标签</Label>
              <Input
                id="import-label"
                placeholder="例如：GitHub / GitLab"
                value={importLabel}
                onChange={(e) => setImportLabel(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="import-pubkey">公钥（必填）</Label>
              <textarea
                id="import-pubkey"
                className="w-full rounded-md border bg-muted/50 p-2 font-mono text-xs"
                rows={3}
                placeholder="ssh-ed25519 AAAA... 或 ssh-rsa AAAA..."
                value={importPubKey}
                onChange={(e) => setImportPubKey(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="import-privkey">私钥（可选）</Label>
              <textarea
                id="import-privkey"
                className="w-full rounded-md border bg-muted/50 p-2 font-mono text-xs"
                rows={5}
                placeholder={"-----BEGIN OPENSSH PRIVATE KEY-----\n..."}
                value={importPrivKey}
                onChange={(e) => setImportPrivKey(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setImportDialogOpen(false)}
            >
              取消
            </Button>
            <Button
              onClick={handleImportSubmit}
              disabled={
                !importLabel.trim() || !importPubKey.trim() || isCreating
              }
            >
              {isCreating ? "保存中..." : "保存"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Created Outbound Key Display Dialog */}
      <Dialog
        open={createdKeyDialogOpen}
        onOpenChange={setCreatedKeyDialogOpen}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>密钥对已生成</DialogTitle>
            <DialogDescription>
              请立即复制公钥并下载私钥。私钥仅在此时可见，关闭后将无法再次查看。
            </DialogDescription>
          </DialogHeader>
          {displayedCreatedKey && (
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label>公钥</Label>
                  <CopyButton
                    text={displayedCreatedKey.public_key}
                    label="公钥已复制"
                  />
                </div>
                <pre className="max-h-24 overflow-auto whitespace-pre-wrap break-all rounded-lg border bg-muted/50 p-3 font-mono text-xs">
                  {displayedCreatedKey.public_key}
                </pre>
                <p className="text-xs text-muted-foreground">
                  将此公钥添加到 GitHub、GitLab 等平台的 SSH Keys 中。
                </p>
              </div>
              {displayedCreatedKey.private_key && (
                <div className="space-y-2">
                  <Label>私钥</Label>
                  <Button
                    variant="outline"
                    size="sm"
                    className="gap-1.5"
                    onClick={() => handleDownloadPrivKey(displayedCreatedKey)}
                  >
                    <Download className="h-3.5 w-3.5" />
                    下载私钥文件
                  </Button>
                </div>
              )}
            </div>
          )}
          <DialogFooter>
            <Button onClick={() => setCreatedKeyDialogOpen(false)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
