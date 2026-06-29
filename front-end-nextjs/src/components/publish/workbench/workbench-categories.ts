import { getHotCategories } from "@/lib/api";

export async function loadWorkbenchCategories() {
  try {
    return await getHotCategories(20);
  } catch {
    return [];
  }
}
