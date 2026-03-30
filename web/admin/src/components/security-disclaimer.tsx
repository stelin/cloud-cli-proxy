import { useState, useEffect } from "react";
import { ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";

const DISCLAIMER_KEY = "security_disclaimer_accepted";
const DISCLAIMER_VERSION = "1";

function hasAccepted(): boolean {
  return localStorage.getItem(DISCLAIMER_KEY) === DISCLAIMER_VERSION;
}

function markAccepted(): void {
  localStorage.setItem(DISCLAIMER_KEY, DISCLAIMER_VERSION);
}

export function SecurityDisclaimer() {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!hasAccepted()) {
      setOpen(true);
    }
  }, []);

  function handleAccept() {
    markAccepted();
    setOpen(false);
  }

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-2xl rounded-2xl bg-background shadow-2xl border overflow-hidden">
        <div className="flex items-center gap-3 border-b px-6 py-5">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10 shrink-0">
            <ShieldCheck className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h2 className="text-lg font-semibold">安全与隐私声明</h2>
            <p className="text-sm text-muted-foreground">使用本平台前，请仔细阅读以下内容</p>
          </div>
        </div>

        <div className="max-h-[65vh] overflow-y-auto px-6 py-5 text-sm leading-relaxed space-y-5">
          <Section title="你的数据对管理员透明">
            <Li>
              你的 <B>SSH 私钥</B>在平台上以明文存储，平台管理员（即宿主机所有者）可以直接查看和获取。
            </Li>
            <Li>
              管理员对你的容器拥有<B>完整的 root 权限</B>，可以随时进入你的容器环境、读取任何文件、查看任何进程。
            </Li>
            <Li>
              你在容器中产生的所有网络流量都经由宿主机路由，理论上可被宿主机截获。
            </Li>
            <Li>
              你的 <B>SSH 密码、登录密码</B>同样由平台管理，管理员可重置和查看。
            </Li>
          </Section>

          <Section title="推荐做法" variant="positive">
            <Li>为本平台生成<B>专用 SSH 密钥</B>，不要使用你在 GitHub、GitLab 或其他重要平台上的主力密钥。</Li>
            <Li>把本平台视为一个<B>公共开发环境</B>，只放你愿意被他人看到的代码和数据。</Li>
            <Li>如果需要向 GitHub 等平台推送代码，使用平台生成的专用密钥，并仅授予最小权限。</Li>
          </Section>

          <Section title="请勿这样做">
            <Li><B>不要</B>在容器中存放生产环境的 API Key、Token、数据库密码或任何关键凭证。</Li>
            <Li><B>不要</B>在容器中登录你的主要银行账户、重要邮箱或其他高敏感在线服务。</Li>
            <Li>
              <B>不要</B>将本地机器上的{" "}
              <Code>~/.ssh/id_rsa</Code> 或 <Code>~/.ssh/id_ed25519</Code>{" "}
              复制到本平台。
            </Li>
            <Li><B>不要</B>把包含敏感信息的 <Code>.env</Code> 文件放入容器。</Li>
          </Section>

          <div className="rounded-lg border bg-muted/40 p-4 text-muted-foreground">
            <p className="font-medium text-foreground mb-1">给新手的提醒</p>
            <p>
              如果你不确定本平台运营者的身份和信誉，请不要将重要的开发环境、私有代码和个人凭证部署到本平台上。
              任何第三方托管的云主机，本质上就是在使用他人的服务器 —— 你的数据安全取决于对运营者的信任。
            </p>
          </div>
        </div>

        <div className="flex items-center justify-between border-t px-6 py-4">
          <p className="text-xs text-muted-foreground">
            点击确认即表示你已阅读并接受上述内容
          </p>
          <Button size="lg" onClick={handleAccept} className="gap-2">
            <ShieldCheck className="h-4 w-4" />
            我已阅读并知晓风险
          </Button>
        </div>
      </div>
    </div>
  );
}

function Section({
  title,
  variant,
  children,
}: {
  title: string;
  variant?: "positive";
  children: React.ReactNode;
}) {
  return (
    <section className="space-y-2">
      <h3
        className={`font-semibold ${variant === "positive" ? "text-emerald-600 dark:text-emerald-400" : "text-foreground"}`}
      >
        {title}
      </h3>
      <ul className="list-disc pl-5 space-y-1.5 text-muted-foreground">
        {children}
      </ul>
    </section>
  );
}

function Li({ children }: { children: React.ReactNode }) {
  return <li>{children}</li>;
}

function B({ children }: { children: React.ReactNode }) {
  return <strong className="text-foreground">{children}</strong>;
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded bg-muted px-1 py-0.5 text-xs">{children}</code>
  );
}
