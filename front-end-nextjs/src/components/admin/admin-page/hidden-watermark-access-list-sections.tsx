"use client";

import { Loader2, Plus, Save, ShieldCheck, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ToggleSwitch } from "./form-fields";
import { AccessList, ManualType } from "./hidden-watermark-access-model";
import { Panel } from "./layout-widgets";
import { AdminObjectPicker } from "./object-picker";
import { LoadingBlock } from "./resource-editor";
import type { HiddenWatermarkAccessController } from "./hidden-watermark-access-view-types";

export function WatermarkGlobalAccessSection({ controller }: { controller: HiddenWatermarkAccessController }) {
  const { allUsers, loading, save, saving, setAllUsers, t, totalAccessCount } = controller;
  return (
      <Panel
        title={t("global.title")}
        icon={ShieldCheck}
        action={
          <Button
            type="button"
            disabled={saving || loading}
            onClick={() => void save()}
            className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"
          >
            {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
            <span>{saving ? t("saving") : t("save")}</span>
          </Button>
        }
      >
        {loading ? (
          <LoadingBlock label={t("loading")} />
        ) : (
          <div className="grid gap-3 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
            <div className="min-w-0 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
              <ToggleSwitch
                value={allUsers}
                onChange={setAllUsers}
                disabled={saving}
                onLabel={t("global.on")}
                offLabel={t("global.off")}
              />
              <p className="mt-2 text-xs leading-5 text-[#737b88]">{t("global.description")}</p>
            </div>
            <div className="grid min-w-0 gap-2 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 text-sm">
              <div className="flex min-w-0 items-center justify-between gap-3">
                <span className="font-semibold text-[#303642]">{t("lists.title")}</span>
                <span className="shrink-0 rounded-full bg-white px-2 py-1 text-xs font-semibold text-[#59606c]">
                  {t("lists.count", { count: totalAccessCount })}
                </span>
              </div>
              <p className="text-xs leading-5 text-[#737b88]">{t("lists.description")}</p>
            </div>
          </div>
        )}
      </Panel>
  );
}

export function WatermarkPickerManualSection({ controller }: { controller: HiddenWatermarkAccessController }) {
  const { addSelectedUsers, loading, manualType, manualValue, saving, selectedUsers, setManualType, setManualValue, setSelectedUsers, submitManual, t, token } = controller;
  return (
      <div className="grid gap-4 xl:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
        <Panel title={t("picker.title")} icon={Users}>
          <div className="grid gap-3">
            <p className="text-sm leading-6 text-[#667085]">{t("picker.description")}</p>
            <AdminObjectPicker
              token={token}
              resource="users"
              label={t("picker.pickerLabel")}
              value={selectedUsers}
              onChange={setSelectedUsers}
              multiple
              placeholder={t("picker.pickerPlaceholder")}
              emptyLabel={t("picker.pickerEmpty")}
              addEmptyLabel={t("picker.addEmpty")}
              clearLabel={t("picker.clear")}
              disabled={loading || saving}
              loadingLabel={t("picker.loading")}
              removeTitle={t("picker.removeSelected")}
              searchLabel={t("picker.search")}
            />
            <Button
              type="button"
              variant="outline"
              disabled={loading || saving || !selectedUsers.length}
              onClick={addSelectedUsers}
              className="h-10 rounded-lg border-black/[0.08] bg-white"
            >
              <Plus className="size-4" />
              <span>{t("picker.addSelected")}</span>
            </Button>
          </div>
        </Panel>
        <Panel title={t("manual.title")} icon={Plus}>
          <form onSubmit={submitManual} className="grid gap-3">
            <p className="text-sm leading-6 text-[#667085]">{t("manual.description")}</p>
            <div className="grid gap-2 sm:grid-cols-[160px_minmax(0,1fr)_auto]">
              <select
                value={manualType}
                onChange={(event) => setManualType(event.target.value as ManualType)}
                disabled={loading || saving}
                className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
              >
                <option value="id">{t("manual.manualTypeId")}</option>
                <option value="username">{t("manual.manualTypeUsername")}</option>
              </select>
              <input
                value={manualValue}
                onChange={(event) => setManualValue(event.target.value)}
                disabled={loading || saving}
                placeholder={t("manual.manualPlaceholder")}
                className="h-10 min-w-0 rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]"
              />
              <Button type="submit" disabled={loading || saving} className="h-10 rounded-lg bg-[#1d4ed8] px-4 hover:bg-[#1e40af]">
                <Plus className="size-4" />
                <span>{t("manual.addManual")}</span>
              </Button>
            </div>
          </form>
        </Panel>
      </div>
  );
}

export function WatermarkAccessListsSection({ controller }: { controller: HiddenWatermarkAccessController }) {
  const { loading, removeValue, t, userIDs, usernames } = controller;
  return (
      <Panel title={t("lists.title")} icon={ShieldCheck}>
        {loading ? (
          <LoadingBlock label={t("loading")} />
        ) : (
          <div className="grid gap-4 lg:grid-cols-2">
            <AccessList
              title={t("lists.userIdsTitle")}
              hint={t("lists.userIdsHint")}
              emptyLabel={t("lists.emptyUserIds")}
              items={userIDs}
              removeLabel={t("lists.remove")}
              onRemove={(value) => removeValue("id", value)}
            />
            <AccessList
              title={t("lists.usernamesTitle")}
              hint={t("lists.usernamesHint")}
              emptyLabel={t("lists.emptyUsernames")}
              items={usernames}
              removeLabel={t("lists.remove")}
              onRemove={(value) => removeValue("username", value)}
            />
          </div>
        )}
      </Panel>
  );
}
