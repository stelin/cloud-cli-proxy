import { cn } from "@/lib/utils";

type StatusDotVariant =
  | "success"
  | "warning"
  | "info"
  | "danger"
  | "muted"
  | "loading";

interface StatusDotProps {
  variant: StatusDotVariant;
  pulse?: boolean;
  size?: "sm" | "md";
  className?: string;
}

const variantClass: Record<StatusDotVariant, string> = {
  success: "bg-success",
  warning: "bg-warning",
  info: "bg-info",
  danger: "bg-destructive",
  muted: "bg-muted-foreground/50",
  loading: "bg-info",
};

export function StatusDot({
  variant,
  pulse = false,
  size = "sm",
  className,
}: StatusDotProps) {
  const sizeClass = size === "md" ? "h-2.5 w-2.5" : "h-2 w-2";

  return (
    <span
      role="presentation"
      className={cn(
        "relative inline-flex shrink-0 items-center justify-center",
        sizeClass,
        className,
      )}
    >
      <span
        className={cn(
          "absolute inset-0 rounded-full",
          variantClass[variant],
        )}
      />
      {pulse ? (
        <span
          className={cn(
            "absolute inset-0 rounded-full opacity-60",
            variantClass[variant],
            "motion-safe:animate-ping motion-reduce:animate-none",
          )}
        />
      ) : null}
    </span>
  );
}
