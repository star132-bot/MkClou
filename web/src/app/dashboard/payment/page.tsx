"use client";

import { CircleAlert, CircleCheck, ExternalLink, LoaderCircle, ShieldCheck } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { toast } from "sonner";

import { FormAlert } from "@/components/auth/auth-shell";
import { Field } from "@/components/auth/form-field";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Segmented } from "@/components/ui/segmented";
import { Textarea } from "@/components/ui/textarea";
import { ApiError } from "@/lib/api/client";
import { formatTime } from "@/lib/format";
import { shopApi } from "@/lib/shop/api";
import type { PaymentCredentials, PaymentEnv, PaymentView } from "@/lib/shop/types";
import { cn } from "@/lib/utils";

const EMPTY: PaymentCredentials = { env: "SANDBOX", appId: "", privateKey: "", publicKey: "" };

type TestState =
  | { kind: "idle" }
  | { kind: "testing" }
  | { kind: "passed"; snapshot: string }
  | { kind: "failed"; message: string; detail?: string };

// 收款设置（PRD SHOP-04）：商家绑定自己的支付宝应用，买家付款直接进入商家账户
export default function PaymentSettingsPage() {
  const [view, setView] = useState<PaymentView | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);

  useEffect(() => {
    shopApi
      .payment()
      .then((v) => {
        setView(v);
        setEditing(!v.configured);
      })
      .catch((e: unknown) => setLoadError(e instanceof ApiError ? e.message : "加载失败，请刷新重试"));
  }, []);

  return (
    <main className="mx-auto w-full max-w-[880px] px-4 pt-8 pb-32 md:px-8">
      <h1 className="text-h1">收款设置</h1>
      <p className="mt-2 flex items-center gap-2">
        <ShieldCheck className="size-4 shrink-0 text-success" strokeWidth={1.5} aria-hidden />
        买家付款直接进入你自己的支付宝账户，MkClou 不经手资金、不抽成。
      </p>

      <div className="mt-8 flex flex-col gap-6">
        {loadError ? (
          <FormAlert>{loadError}</FormAlert>
        ) : !view ? (
          <div className="flex flex-col gap-6" aria-busy="true">
            <div className="h-28 animate-pulse rounded-lg bg-muted" />
            <div className="h-96 animate-pulse rounded-lg bg-muted" />
          </div>
        ) : (
          <>
            <StatusCard view={view} editing={editing} onEdit={() => setEditing(true)} />
            {editing && (
              <>
                <GuideCard />
                <CredentialsForm
                  configured={view.configured}
                  initialEnv={view.env || "SANDBOX"}
                  onCancel={view.configured ? () => setEditing(false) : undefined}
                  onSaved={(v) => {
                    setView(v);
                    setEditing(false);
                  }}
                />
              </>
            )}
          </>
        )}
      </div>
    </main>
  );
}

function Card({ title, description, children }: { title: string; description?: React.ReactNode; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-6 rounded-lg border border-border bg-background p-6">
      <div>
        <h2 className="text-h3">{title}</h2>
        {description && <p className="mt-1 text-caption text-text-tertiary">{description}</p>}
      </div>
      {children}
    </section>
  );
}

function StatusCard({ view, editing, onEdit }: { view: PaymentView; editing: boolean; onEdit: () => void }) {
  if (!view.configured) {
    return (
      <section className="flex items-start gap-3 rounded-lg border border-border bg-background p-6">
        <CircleAlert className="mt-0.5 size-5 shrink-0 text-warning" strokeWidth={1.5} aria-hidden />
        <div>
          <h2 className="text-h3">尚未配置收款</h2>
          <p className="mt-1">配置完成前，付费商品无法上架；免费商品不受影响。</p>
        </div>
      </section>
    );
  }

  const active = view.status === "ACTIVE";
  const sandbox = view.env === "SANDBOX";
  return (
    <section className="flex flex-col gap-4 rounded-lg border border-border bg-background p-6 sm:flex-row sm:items-start">
      {active ? (
        <CircleCheck className="mt-0.5 size-5 shrink-0 text-success" strokeWidth={1.5} aria-hidden />
      ) : (
        <CircleAlert className="mt-0.5 size-5 shrink-0 text-danger" strokeWidth={1.5} aria-hidden />
      )}
      <div className="min-w-0 flex-1">
        <h2 className="flex flex-wrap items-center gap-2 text-h3">
          {active ? "收款已生效" : "收款配置已失效"}
          <span
            className={cn(
              "rounded-sm px-2 py-0.5 text-caption font-medium",
              sandbox ? "bg-warning/10 text-warning" : "bg-success/10 text-success",
            )}
          >
            {sandbox ? "沙箱环境" : "正式环境"}
          </span>
        </h2>
        {sandbox && active && <p className="mt-1">当前为测试模式：买家看到“测试模式”提示，付款不会产生真实扣费。</p>}
        {!active && view.lastError && <p className="mt-1 text-danger">{view.lastError}</p>}
        <dl className="mt-4 grid gap-x-6 gap-y-2 sm:grid-cols-[auto_1fr]">
          <dt className="text-text-tertiary">AppID</dt>
          <dd className="font-mono text-text-primary">{view.appId}</dd>
          <dt className="text-text-tertiary">应用私钥</dt>
          <dd className="text-text-primary">
            已加密保存（末尾 <span className="font-mono">{view.privateKeyTail}</span>）
          </dd>
          {view.verifiedAt && (
            <>
              <dt className="text-text-tertiary">最近验证</dt>
              <dd className="text-text-primary">{formatTime(view.verifiedAt)}</dd>
            </>
          )}
        </dl>
      </div>
      {!editing && (
        <Button variant="outline" onClick={onEdit}>
          {sandbox ? "切换到正式环境 / 更新" : "更新配置"}
        </Button>
      )}
    </section>
  );
}

function GuideCard() {
  return (
    <Card title="如何获取参数" description="首次配置约需 10 分钟。建议先用沙箱环境跑通，再切换到正式环境。">
      <ol className="flex list-decimal flex-col gap-3 pl-5">
        <li>
          登录
          <a href="https://open.alipay.com" target="_blank" rel="noopener noreferrer" className="mx-1 inline-flex items-center gap-1 text-brand underline-offset-4 hover:underline">
            支付宝开放平台
            <ExternalLink className="size-3.5" strokeWidth={1.5} aria-hidden />
          </a>
          。测试时进入「沙箱环境」即可直接使用沙箱应用；正式收款需创建网页 / 移动应用并签约「电脑网站支付」「手机网站支付」。
        </li>
        <li>在应用的「开发设置」中，接口加签方式选择「公钥」，用支付宝密钥工具生成密钥，上传应用公钥。</li>
        <li>复制三项内容到下方：应用的 AppID、你生成的「应用私钥」、上传后平台显示的「支付宝公钥」。</li>
        <li>点击「测试连接」，通过后保存。</li>
      </ol>
      <p className="rounded-md bg-surface p-3 text-caption text-text-tertiary">
        注意：「支付宝公钥」和「应用公钥」不是同一个。应用私钥只会以加密形式保存，保存后不会再显示。
      </p>
    </Card>
  );
}

function CredentialsForm({
  configured,
  initialEnv,
  onCancel,
  onSaved,
}: {
  configured: boolean;
  initialEnv: PaymentEnv;
  onCancel?: () => void;
  onSaved: (v: PaymentView) => void;
}) {
  const envLabelId = useId();
  const [values, setValues] = useState<PaymentCredentials>({ ...EMPTY, env: initialEnv });
  const [errors, setErrors] = useState<Partial<Record<keyof PaymentCredentials, string>>>({});
  const [test, setTest] = useState<TestState>({ kind: "idle" });
  const [saving, setSaving] = useState(false);

  const snapshot = JSON.stringify(values);
  // 测试通过后修改了任何字段，需要重新测试（保存时服务端也会再测一次）
  const passed = test.kind === "passed" && test.snapshot === snapshot;

  const set = <K extends keyof PaymentCredentials>(k: K, v: PaymentCredentials[K]) => {
    setValues((s) => ({ ...s, [k]: v }));
    setErrors((e) => ({ ...e, [k]: undefined }));
  };

  const validate = (): boolean => {
    const next: typeof errors = {};
    if (!/^\d{16}$/.test(values.appId.trim())) next.appId = "AppID 为 16 位数字";
    if (values.privateKey.replace(/\s/g, "").length < 100) next.privateKey = "请粘贴完整的应用私钥";
    if (values.publicKey.replace(/\s/g, "").length < 100) next.publicKey = "请粘贴完整的支付宝公钥";
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const applyError = (e: unknown): string => {
    if (!(e instanceof ApiError)) return "网络异常，请稍后重试";
    const fields = e.fields as Partial<Record<keyof PaymentCredentials, string>>;
    if (Object.keys(fields).length > 0) setErrors(fields);
    return e.message;
  };

  const runTest = async () => {
    if (!validate()) return;
    setTest({ kind: "testing" });
    try {
      await shopApi.testPayment(values);
      setTest({ kind: "passed", snapshot });
    } catch (e) {
      const detail = e instanceof ApiError ? (e.data as { detail?: string } | null)?.detail : undefined;
      setTest({ kind: "failed", message: applyError(e), detail });
    }
  };

  const save = async () => {
    setSaving(true);
    try {
      const v = await shopApi.savePayment(values);
      toast.success(v.env === "SANDBOX" ? "已保存，当前为沙箱测试模式" : "已保存，可以开始收款了");
      onSaved(v);
    } catch (e) {
      setTest({ kind: "failed", message: applyError(e) });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card title={configured ? "更新收款配置" : "填写收款参数"} description="出于安全考虑，更新时需要重新填写全部参数。">
      <div className="flex flex-col gap-2">
        <span id={envLabelId} className="text-body font-medium text-text-primary">
          环境
        </span>
        <Segmented<PaymentEnv>
          aria-labelledby={envLabelId}
          value={values.env}
          onValueChange={(v) => set("env", v)}
          options={[
            { value: "SANDBOX", label: "沙箱（测试）" },
            { value: "PRODUCTION", label: "正式" },
          ]}
          className="w-fit"
        />
        <p className="text-caption text-text-tertiary">
          {values.env === "SANDBOX" ? "沙箱付款不会产生真实扣费，买家页面会显示“测试模式”。" : "正式环境下买家付款会真实扣费，款项直接进入你的支付宝账户。"}
        </p>
      </div>

      <Field label="AppID" error={errors.appId} hint="开放平台应用详情页中的 16 位数字">
        {({ id, describedBy, invalid }) => (
          <Input
            id={id}
            value={values.appId}
            onChange={(e) => set("appId", e.target.value)}
            inputMode="numeric"
            autoComplete="off"
            spellCheck={false}
            maxLength={20}
            className="font-mono"
            aria-describedby={describedBy}
            aria-invalid={invalid}
          />
        )}
      </Field>

      <Field label="应用私钥" error={errors.privateKey} hint="密钥工具生成的“应用私钥”，可以包含 BEGIN / END 行">
        {({ id, describedBy, invalid }) => (
          <Textarea
            id={id}
            value={values.privateKey}
            onChange={(e) => set("privateKey", e.target.value)}
            rows={5}
            autoComplete="off"
            spellCheck={false}
            className="font-mono text-caption break-all"
            aria-describedby={describedBy}
            aria-invalid={invalid}
          />
        )}
      </Field>

      <Field label="支付宝公钥" error={errors.publicKey} hint="上传应用公钥后，开放平台显示的“支付宝公钥”">
        {({ id, describedBy, invalid }) => (
          <Textarea
            id={id}
            value={values.publicKey}
            onChange={(e) => set("publicKey", e.target.value)}
            rows={4}
            autoComplete="off"
            spellCheck={false}
            className="font-mono text-caption break-all"
            aria-describedby={describedBy}
            aria-invalid={invalid}
          />
        )}
      </Field>

      <div aria-live="polite">
        {test.kind === "passed" && passed && (
          <FormAlert tone="success">测试通过：AppID、应用私钥与支付宝公钥均正确。</FormAlert>
        )}
        {test.kind === "passed" && !passed && <FormAlert tone="info">参数已修改，请重新测试连接。</FormAlert>}
        {test.kind === "failed" && (
          <FormAlert>
            {test.message}
            {test.detail && <span className="mt-1 block font-mono text-caption opacity-80">错误码：{test.detail}</span>}
          </FormAlert>
        )}
      </div>

      <div className="flex flex-wrap gap-3 border-t border-border pt-6">
        <Button variant="outline" onClick={runTest} disabled={test.kind === "testing" || saving}>
          {test.kind === "testing" && <LoaderCircle className="animate-spin" strokeWidth={1.5} data-icon="inline-start" />}
          {test.kind === "testing" ? "测试中…" : "测试连接"}
        </Button>
        <Button onClick={save} disabled={!passed || saving} title={passed ? undefined : "请先测试连接"}>
          {saving ? "保存中…" : "保存"}
        </Button>
        {onCancel && (
          <Button variant="ghost" onClick={onCancel} disabled={saving}>
            取消
          </Button>
        )}
      </div>
    </Card>
  );
}
