import { describe, it, expect } from "vitest";
import { isFeaturedImageEmbedded } from "../lib/featuredImage";

const PEXELS = "https://images.pexels.com/photos/12345/pexels-photo-12345.jpeg";

describe("isFeaturedImageEmbedded", () => {
  it("returns false when featured_image_url is missing", () => {
    expect(isFeaturedImageEmbedded(undefined, `<img src="${PEXELS}" />`)).toBe(
      false,
    );
    expect(isFeaturedImageEmbedded("", `<img src="${PEXELS}" />`)).toBe(false);
  });

  it("returns false when content is missing", () => {
    expect(isFeaturedImageEmbedded(PEXELS, undefined)).toBe(false);
    expect(isFeaturedImageEmbedded(PEXELS, "")).toBe(false);
  });

  it("detects the same image embedded in an HTML figure", () => {
    const content = `<p>Intro</p>\n\n<figure><img src="${PEXELS}" alt="AI tools" loading="lazy" /><figcaption>Photo by Matheus Bertelli on Pexels</figcaption></figure>`;
    expect(isFeaturedImageEmbedded(PEXELS, content)).toBe(true);
  });

  it("detects the same image with a different query string", () => {
    const featured = `${PEXELS}?auto=compress&cs=tinysrgb&w=1200`;
    const content = `<figure><img src="${PEXELS}?auto=compress&cs=tinysrgb&w=600" /></figure>`;
    expect(isFeaturedImageEmbedded(featured, content)).toBe(true);
  });

  it("detects the same image with HTML-escaped attribute", () => {
    const escaped = `${PEXELS}?auto=compress&amp;cs=tinysrgb&amp;w=600`;
    const content = `<figure><img src="${escaped}" /></figure>`;
    expect(isFeaturedImageEmbedded(`${PEXELS}?auto=compress&cs=tinysrgb&w=600`, content)).toBe(true);
  });

  it("detects the same image in markdown syntax", () => {
    const content = `![AI tools](${PEXELS})`;
    expect(isFeaturedImageEmbedded(PEXELS, content)).toBe(true);
  });

  it("detects the same image with single-quoted attribute", () => {
    const content = `<figure><img src='${PEXELS}' /></figure>`;
    expect(isFeaturedImageEmbedded(PEXELS, content)).toBe(true);
  });

  it("does not match a different image in the content", () => {
    const other = "https://images.pexels.com/photos/99999/pexels-photo-99999.jpeg";
    const content = `<figure><img src="${other}" /></figure>`;
    expect(isFeaturedImageEmbedded(PEXELS, content)).toBe(false);
  });

  it("does not match by file name alone", () => {
    const other = "https://cdn.example.com/media/pexels-photo-12345.jpeg";
    const content = `<img src="${other}" />`;
    expect(isFeaturedImageEmbedded(PEXELS, content)).toBe(false);
  });

  it("returns false when content has no images", () => {
    expect(isFeaturedImageEmbedded(PEXELS, "<p>Just text, no image.</p>")).toBe(
      false,
    );
  });

  it("normalizes trailing slash differences", () => {
    const content = `<img src="https://example.com/media/photo.jpg/" />`;
    expect(
      isFeaturedImageEmbedded("https://example.com/media/photo.jpg", content),
    ).toBe(true);
  });
});