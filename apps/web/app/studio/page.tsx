import type { Metadata } from "next";
import { Studio } from "@/components/studio";
export const metadata: Metadata = {
  title: "پنل نویسندگان",
  robots: { index: false, follow: false },
};
export default function Page() {
  return <Studio />;
}
