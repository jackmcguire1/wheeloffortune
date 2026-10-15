import { timestampDate, type Timestamp } from "@bufbuild/protobuf/wkt";

const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

export function timeAgo(ts?: Timestamp): string {
  if (!ts) return "";
  const secs = Math.round((timestampDate(ts).getTime() - Date.now()) / 1000);
  const abs = Math.abs(secs);
  if (abs < 60) return rtf.format(secs, "second");
  if (abs < 3600) return rtf.format(Math.round(secs / 60), "minute");
  if (abs < 86400) return rtf.format(Math.round(secs / 3600), "hour");
  return rtf.format(Math.round(secs / 86400), "day");
}
