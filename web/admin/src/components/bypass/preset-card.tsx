import { Lock } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { BypassPreset } from "@/lib/api/types/bypass";

interface PresetCardProps {
  preset: BypassPreset;
  selected: boolean;
  forced: boolean; // loopback 强制锁定
  onToggle: (next: boolean) => void;
  disabled?: boolean;
}

/**
 * 单张预设卡片：
 * - forced（loopback）：实线主色边框 + 主色 5% 背景 + Lock 图标 + checkbox disabled checked
 * - selected：主色边框 + 主色 5% 背景
 * - unselected：默认 border + card 背景
 * - hover 显示样例规则（Tooltip 兜底 Popover 需求）
 */
export function PresetCard({
  preset,
  selected,
  forced,
  onToggle,
  disabled,
}: PresetCardProps) {
  const isActive = forced || selected;
  const ringClass = isActive
    ? "border-primary bg-primary/5"
    : "border-border bg-card";

  // WR-08：后端 BypassPreset 没有 sample_rules / rule_count 字段，直接读 preset.rules
  // 并在前端截取前 3 条用于 Tooltip 展示；卡片副文案用规则总数。
  const ruleCount = preset.rules?.length ?? 0;
  const sampleRules = preset.rules?.slice(0, 3) ?? [];
  const sampleText =
    sampleRules.length > 0
      ? sampleRules.map((s) => `${s.rule_type} · ${s.value}`).join("\n")
      : "暂无规则示例";

  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Card
            data-testid={`preset-card-${preset.slug}`}
            data-state={
              forced ? "forced-on" : selected ? "selected" : "unselected"
            }
            className={`h-24 cursor-pointer border ${ringClass} transition-colors ${
              disabled ? "opacity-50" : ""
            }`}
            onClick={() => {
              if (forced || disabled) return;
              onToggle(!selected);
            }}
          >
            <CardContent className="flex h-full items-start gap-3 p-4">
              <input
                type="checkbox"
                aria-label={`${preset.slug} 预设`}
                checked={isActive}
                disabled={forced || disabled}
                onChange={(e) => {
                  if (forced || disabled) return;
                  onToggle(e.target.checked);
                }}
                onClick={(e) => e.stopPropagation()}
                className="mt-0.5 size-4 accent-primary"
              />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5">
                  <span className="truncate text-base font-semibold">
                    {preset.name || preset.slug}
                  </span>
                  {forced && (
                    <Lock
                      className="size-3 text-muted-foreground"
                      aria-label="强制启用，不可关闭"
                    />
                  )}
                  {forced && (
                    <Badge
                      variant="secondary"
                      className="px-1.5 py-0 text-[10px]"
                    >
                      强制
                    </Badge>
                  )}
                </div>
                <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                  {preset.description ||
                    `共 ${ruleCount} 条规则`}
                </p>
              </div>
            </CardContent>
          </Card>
        </TooltipTrigger>
        <TooltipContent side="bottom" className="max-w-xs whitespace-pre-line text-left">
          <div className="font-semibold">包含的规则</div>
          <div className="mt-1 font-mono text-[11px] leading-relaxed">
            {sampleText}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
