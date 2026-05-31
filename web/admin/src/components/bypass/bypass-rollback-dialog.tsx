import { toast } from "sonner";
import { Loader2, AlertTriangle } from "lucide-react";
import { useRollbackBypass } from "@/hooks/use-bypass-snapshots";
import { parseBypassError } from "@/lib/i18n/bypass-error-codes";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";

interface BypassRollbackDialogProps {
	hostId: string;
	snapshotId: string;
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onRollback: () => void;
}

export function BypassRollbackDialog({
	hostId,
	snapshotId,
	open,
	onOpenChange,
	onRollback,
}: BypassRollbackDialogProps) {
	const rollback = useRollbackBypass(hostId);

	function handleRollback() {
		rollback.mutate(snapshotId, {
			onSuccess: (data) => {
				toast.success(data.message || "回滚成功");
				onOpenChange(false);
				onRollback();
			},
			onError: (err) => {
				toast.error(parseBypassError(err).message);
			},
		});
	}

	return (
		<AlertDialog open={open} onOpenChange={onOpenChange}>
			<AlertDialogContent>
				<AlertDialogHeader>
					<AlertDialogTitle>确认回滚旁路配置？</AlertDialogTitle>
					<AlertDialogDescription>
						将旁路配置回滚到快照 <code className="rounded bg-muted px-1 py-0.5 text-xs">{snapshotId}</code>。回滚后当前未生效的修改将被丢弃，主机上的白名单规则将恢复到该快照对应的状态。
					</AlertDialogDescription>
				</AlertDialogHeader>
				<AlertDialogFooter>
					<AlertDialogCancel disabled={rollback.isPending}>
						取消
					</AlertDialogCancel>
					<AlertDialogAction
						variant="destructive"
						disabled={rollback.isPending}
						onClick={handleRollback}
					>
						{rollback.isPending ? (
							<>
								<Loader2 className="mr-1 size-4 animate-spin" />
								回滚中...
							</>
						) : (
							<>
								<AlertTriangle className="mr-1 size-4" />
								确认回滚
							</>
						)}
					</AlertDialogAction>
				</AlertDialogFooter>
			</AlertDialogContent>
		</AlertDialog>
	);
}
