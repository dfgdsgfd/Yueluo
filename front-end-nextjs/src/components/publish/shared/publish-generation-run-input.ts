export type PublishGenerationRunOptions = {
  fresh?: boolean;
};

export function publishGenerationRunInput<T extends { body: string; title: string }>(
  input: T,
  options?: PublishGenerationRunOptions,
): T {
  if (!options?.fresh) return input;
  return {
    ...input,
    body: "",
    title: "",
  };
}
