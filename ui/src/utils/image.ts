/**
 * Splits a Docker image reference (e.g. "repo/name:tag@sha256:...")
 * into its "image:tag" name part and its digest/SHA part.
 */
export function splitImageReference(image?: string, fallbackSha?: string): { name: string; sha: string } {
  if (!image) {
    return {
      name: '—',
      sha: fallbackSha || '—',
    };
  }

  const atIndex = image.indexOf('@');
  if (atIndex === -1) {
    return {
      name: image,
      sha: fallbackSha || '—',
    };
  }

  const name = image.slice(0, atIndex) || '—';
  const sha = image.slice(atIndex + 1) || fallbackSha || '—';
  return { name, sha };
}
