import { CheckCircle2, XCircle, Loader2, ArrowRight, AlertTriangle } from "lucide-react";
import { useBypassConsistency } from "@/hooks/use-bypass-snapshots";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { BypassConsistencyDiff } from "@/lib/api/types/bypass";

interface BypassConsistencyCardProps {
	hostId: string;
}

function DiffRow({ diff }: { diff: BypassConsistencyDiff }) {
	return (
		<div className="rounded-md border bg-muted/30 px-3 py-2">
			<div className="mb-1 flex items-center gap-1.5">
				<Badge variant="outline" className="text-[10px]">
					{diff.section}
				</Badge>
				<span className="text-xs font-medium">{diff.field}</span>
			</div>
			<div className="flex items-center gap-2 text-xs">
				<span className="min-w-0 flex-1 break-all rounded bg-red-50 px-2 py-0.5 text-red-700 line-through dark:bg-red-950 dark:text-red-400">
					{diff.actual_value}
				</span>
				<ArrowRight className="size-3 shrink-0 text-muted-foreground" />
				<span className="min-w-0 flex-1 break-all rounded bg-green-50 px-2 py-0.5 text-green-700 dark:bg-green-950 dark:text-green-400">
					{diff.expected_value}
				</span>
			</div>
		</div>
	);
}

export function BypassConsistencyCard({ hostId }: BypassConsistencyCardProps) {
	const query = useBypassConsistency(hostId);

	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex items-center gap-2 text-sm font-medium">
					一致性校验
					{query.isFetching && !query.isLoading && (
						<Loader2 className="size-3.5 animate-spin text-muted-foreground" />
					)}
				</CardTitle>
			</CardHeader>
			<CardContent>
				{query.isLoading ? (
					<div className="flex items-center justify-center py-8">
						<Loader2 className="size-4 animate-spin text-muted-foreground" />
						<span className="ml-2 text-sm text-muted-foreground">
							校验中...
						</span>
					</div>
				) : query.isError ? (
					<div className="flex items-center justify-center gap-2 py-6 text-sm text-muted-foreground">
						<AlertTriangle className="size-4 text-amber-500" />
						校验失败，请稍后重试
					</div>
				) : query.data ? (
					<div className="space-y-3">
						<div className="flex items-center gap-2">
							{query.data.is_consistent ? (
								<>
									<CheckCircle2 className="size-5 text-green-600" />
									<Badge
										variant="outline"
										className="border-green-300 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
									>
										一致
									</Badge>
								</>
							) : (
								<>
									<XCircle className="size-5 text-red-600" />
									<Badge
										variant="outline"
										className="border-red-300 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
									>
										不一致
									</Badge>
								</>
							)}
							<span className="text-xs text-muted-foreground">
								校验时间：{query.data.checked_at
									? new Date(query.data.checked_at).toLocaleString("zh-CN")
									: "-"}
							</span>
						</div>

						{query.data.is_consistent ? (
							<p className="text-sm text-muted-foreground">
								预期配置与实际生效配置完全一致，旁路规则已正确下发到主机。
							</p>
						) : (
							<>
								<p className="text-sm text-muted-foreground">
									预期配置与实际生效配置不一致，以下为差异项：
								</p>
								{(query.data.diffs?.length ?? 0) > 0 ? (
									<div className="space-y-2">
										{query.data.diffs!.map((diff, idx) => (
											<DiffRow key={idx} diff={diff} />
										))}
									</div>
								) : (
									<p className="text-sm text-muted-foreground">暂无差异详情</p>
								)}
							</>
						)}
					</div>
				) : null}
			</CardContent>
		</Card>
	);
}
