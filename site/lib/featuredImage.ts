// Featured-image duplication guard.
//
// Auto-generated articles may carry the same image twice: once embedded in the
// body (as <figure><img> or a markdown image) and once as featured_image_url.
// Rendering both on the article page shows the image twice. This helper
// detects whether the featured image is already present inside the content so
// the page can skip the separate hero image without touching the stored data.

// Decodes HTML entities that may appear inside attribute values
// (&amp; last so "&amp;lt;" correctly becomes "<").
function decodeHtmlEntities(input: string): string {
  return input
    .replace(/&#x([0-9a-f]+);/gi, (_, hex: string) =>
      String.fromCodePoint(parseInt(hex, 16)),
    )
    .replace(/&#(\d+);/g, (_, dec: string) =>
      String.fromCodePoint(parseInt(dec, 10)),
    )
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&amp;/g, "&");
}

// Safely percent-decodes a URL; invalid sequences keep the raw value.
function decodePercent(input: string): string {
  try {
    return decodeURIComponent(input);
  } catch {
    return input;
  }
}

// Normalizes a raw URL found in the content or the featured field:
// trims, decodes HTML entities and percent-encoding, and canonicalizes
// scheme/host casing via URL parsing when the URL is absolute.
function normalizeUrl(raw: string): string {
  let url = decodeHtmlEntities(raw.trim());
  url = decodePercent(url);
  try {
    return new URL(url).href;
  } catch {
    return url;
  }
}

// Compares by full path (scheme + host + path), ignoring query strings and
// trailing slashes — the same image file served with different sizing
// parameters (e.g. Pexels ?auto=compress&w=600) is still the same image.
// Never compares by file name alone.
function baseUrl(url: string): string {
  const idx = url.search(/[?#]/);
  return (idx === -1 ? url : url.slice(0, idx)).replace(/\/+$/, "");
}

// Extracts image URLs from a content string that mixes HTML and markdown:
// <img src="..."> (double/single/unquoted) and ![alt](url).
function extractImageUrls(content: string): string[] {
  const urls: string[] = [];
  const imgTagRe = /<img\b[^>]*?\bsrc\s*=\s*("([^"]*)"|'([^']*)'|([^\s"'<>`]+))/gi;
  let m: RegExpExecArray | null;
  while ((m = imgTagRe.exec(content)) !== null) {
    urls.push(m[2] ?? m[3] ?? m[4] ?? "");
  }
  const markdownRe = /!\[[^\]]*\]\(\s*([^)\s]+)\s*\)/g;
  while ((m = markdownRe.exec(content)) !== null) {
    urls.push(m[1]);
  }
  return urls;
}

// Reports whether the featured image URL already appears inside the content
// HTML/markdown. Returns false when either value is missing.
export function isFeaturedImageEmbedded(
  featuredUrl: string | undefined,
  content: string | undefined,
): boolean {
  if (!featuredUrl || !content) {
    return false;
  }
  const target = normalizeUrl(featuredUrl);
  if (!target) {
    return false;
  }
  const targetBase = baseUrl(target);
  for (const raw of extractImageUrls(content)) {
    if (!raw) {
      continue;
    }
    const norm = normalizeUrl(raw);
    if (!norm) {
      continue;
    }
    if (norm === target || baseUrl(norm) === targetBase) {
      return true;
    }
  }
  return false;
}