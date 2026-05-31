import { useState, useEffect } from "react";
import { FileText, Loader2, ChevronDown, User } from "lucide-react";
import { useBypassAuditLog } from "@/hooks/use-bypass-snapshots";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { BypassAuditLogEntry } from "@/lib/api/types/bypass";

interface BypassAuditPanelProps {
	hostId: string;
}

const ACTION_LABELS: Record<string, string> = {
	apply: "应用配置",
	rollback: "回滚配置",
	create_rule: "创建规则",
	update_rule: "更新规则",
	delete_rule: "删除规则",
	create_binding: "绑定预设",
	delete_binding: "解绑预设",
};

function actionLabel(action: string): string {
	return ACTION_LABELS[action] ?? action;
}

function formatTime(iso: string): string {
	try {
		const d = new Date(iso);
		return d.toLocaleString("zh-CN", {
			year: "numeric",
			month: "2-digit",
			day: "2-digit",
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
		});
	} catch {
		return iso;
	}
}

export function BypassAuditPanel({ hostId }: BypassAuditPanelProps) {
	const [cursor, setCursor] = useState<string | undefined>();
	const [allEntries, setAllEntries] = useState<BypassAuditLogEntry[]>([]);

	const query = useBypassAuditLog(hostId, cursor);

	useEffect(() => {
		if (query.data?.audit_log) {
			setAllEntries((prev) => {
				const existingIds = new Set(prev.map((e) => e.id));
				const newEntries = query.data.audit_log.filter(
					(e) => !existingIds.has(e.id),
				);
				return [...prev, ...newEntries];
			});
		}
	}, [query.data]);

	const hasMore =
		query.data?.next_before != null && query.data.next_before !== "";
	const isLoading = query.isLoading && allEntries.length === 0;

	function handleLoadMore() {
		if (query.data?.next_before) {
			setCursor(query.data.next_before);
		}
	}

	const entries = allEntries;

	return (
		<Card>
			<CardHeader>
				<CardTitle className="text-sm font-medium">操作审计</CardTitle>
			</CardHeader>
			<CardContent>
				{isLoading ? (
					<div className="flex items-center justify-center py-8">
						<Loader2 className="size-4 animate-spin text-muted-foreground" />
						<span className="ml-2 text-sm text-muted-foreground">
							加载中...
						</span>
					</div>
				) : entries.length === 0 ? (
					<p className="py-8 text-center text-sm text-muted-foreground">
						暂无操作记录
					</p>
				) : (
					<div className="relative space-y-0">
						<div className="absolute left-[11px] top-1 bottom-1 w-px bg-border" />
						{entries.map((entry) => (
							<div
								key={entry.id}
								className="relative flex gap-3 pb-4 last:pb-0"
							>
								<div className="relative z-10 mt-1.5 size-2.5 rounded-full border-2 border-border bg-background" />
								<div className="min-w-0 flex-1 space-y-1">
									<div className="flex items-center gap-2">
										<Badge variant="secondary" className="text-[10px]">
											{actionLabel(entry.action)}
										</Badge>
										<span className="text-xs text-muted-foreground">
											{formatTime(entry.created_at)}
										</span>
									</div>
									{entry.note && (
										<div className="flex items-start gap-1 text-xs text-muted-foreground">
											<FileText className="mt-0.5 size-3 shrink-0" />
											<span className="break-all">{entry.note}</span>
										</div>
									)}
									<div className="flex items-center gap-1 text-xs text-muted-foreground">
										<User className="size-3 shrink-0" />
										<span>{entry.actor_ip}</span>
									</div>
								</div>
							</div>
						))}
						{hasMore && (
							<div className="relative flex justify-center pt-2">
								<Button
									variant="ghost"
									size="sm"
									onClick={handleLoadMore}
									disabled={query.isFetching}
								>
									{query.isFetching ? (
										<>
											<Loader2 className="mr-1 size-3 animate-spin" />
											加载中...
										</>
									) : (
										<>
											<ChevronDown className="mr-1 size-3" />
											加载更多
										</>
									)}
								</Button>
							</div>
						)}
					</div>
				)}
			</CardContent>
		</Card>
	);
}
