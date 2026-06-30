"use client";

import { FileText } from "lucide-react";

export type SystemNotificationTemplate = {
  content: string;
  label: string;
  showPopup: boolean;
  title: string;
  type: string;
};

export const systemNotificationTemplates: SystemNotificationTemplate[] = [
  {
    label: "维护通知",
    title: "系统维护通知",
    type: "system",
    showPopup: true,
    content: [
      "## 系统维护通知",
      "",
      "为提升服务稳定性，平台将进行短时维护。",
      "",
      "- **维护时间**：YYYY-MM-DD HH:mm - HH:mm",
      "- **影响范围**：登录、发布、消息通知可能短暂不可用",
      "- **预计恢复**：维护完成后自动恢复，无需重新安装客户端",
      "",
      "> 如维护提前结束，我们会第一时间恢复服务。",
    ].join("\n"),
  },
  {
    label: "版本更新",
    title: "新版本功能更新",
    type: "update",
    showPopup: true,
    content: [
      "## 新版本功能更新",
      "",
      "本次更新已上线，建议刷新页面或重启 App 后体验。",
      "",
      "### 更新内容",
      "",
      "1. 优化消息通知与系统弹窗体验",
      "2. 修复部分页面在移动端的显示问题",
      "3. 提升图片与视频加载稳定性",
      "",
      "[查看详情](/notifications)",
    ].join("\n"),
  },
  {
    label: "活动公告",
    title: "限时活动开启",
    type: "activity",
    showPopup: true,
    content: [
      "## 限时活动开启",
      "",
      "活动期间完成指定任务即可获得额外奖励。",
      "",
      "| 任务 | 奖励 | 说明 |",
      "| --- | --- | --- |",
      "| 完善资料 | 积分奖励 | 每位用户限一次 |",
      "| 发布内容 | 成长奖励 | 需通过审核 |",
      "| 互动参与 | 活跃奖励 | 每日统计 |",
      "",
      "**活动时间**：YYYY-MM-DD 至 YYYY-MM-DD",
    ].join("\n"),
  },
  {
    label: "安全提醒",
    title: "账号安全提醒",
    type: "warning",
    showPopup: true,
    content: [
      "## 账号安全提醒",
      "",
      "近期请注意保护账号与个人信息安全。",
      "",
      "- 不要向他人透露验证码、密码或登录链接",
      "- 遇到异常登录提醒，请及时修改密码",
      "- 平台工作人员不会索要你的密码或验证码",
      "",
      "> 如发现异常，请尽快联系平台支持。",
    ].join("\n"),
  },
  {
    label: "奖励发放",
    title: "奖励已发放",
    type: "system",
    showPopup: false,
    content: [
      "## 奖励已发放",
      "",
      "你参与的活动奖励已完成发放。",
      "",
      "- **奖励类型**：积分 / 余额 / 礼品卡",
      "- **发放时间**：YYYY-MM-DD HH:mm",
      "- **查看入口**：[前往钱包](/wallet)",
      "",
      "感谢你的参与。",
    ].join("\n"),
  },
];

export function SystemNotificationTemplatePicker({
  onApply,
}: {
  onApply: (template: SystemNotificationTemplate) => void;
}) {
  return (
    <section className="rounded-lg border border-[#1d4ed8]/15 bg-[#eff6ff] p-3">
      <div className="flex items-center gap-2 text-sm font-semibold text-[#1e3a8a]">
        <FileText className="size-4" />
        <span>常用 Markdown 模板</span>
      </div>
      <p className="mt-1 text-xs leading-5 text-[#526178]">一键填充标题、类型、弹窗开关和正文，保存前可继续编辑。</p>
      <div className="mt-3 grid gap-2 sm:grid-cols-2">
        {systemNotificationTemplates.map((template) => (
          <button
            key={template.label}
            type="button"
            onClick={() => onApply(template)}
            className="rounded-lg border border-white bg-white px-3 py-2 text-left text-sm font-semibold text-[#263246] shadow-sm transition hover:border-[#93c5fd] hover:bg-[#f8fbff] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/25"
          >
            <span className="block truncate">{template.label}</span>
            <span className="mt-0.5 block text-xs font-medium text-[#6b7280]">{template.title}</span>
          </button>
        ))}
      </div>
    </section>
  );
}
