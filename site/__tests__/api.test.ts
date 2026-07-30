import type { Article, Category } from "../lib/api";

describe("API Types", () => {
  it("should validate Article type structure", () => {
    const article: Article = {
      id: "123e4567-e89b-12d3-a456-426614174000",
      site_id: "123e4567-e89b-12d3-a456-426614174001",
      title: "Test Article",
      slug: "test-article",
      excerpt: "A test article",
      featured_image_url: "https://example.com/image.jpg",
      published_at: "2026-07-30T10:00:00Z",
      language: "pt",
      tags: ["tech", "cms"],
      categories: ["Tecnologia"],
      word_count: 500,
      reading_time: 3,
    };
    expect(article.title).toBe("Test Article");
    expect(article.slug).toBe("test-article");
    expect(article.word_count).toBe(500);
    expect(article.reading_time).toBe(3);
    expect(article.language).toBe("pt");
    expect(article.tags).toHaveLength(2);
  });

  it("should accept optional fields as undefined", () => {
    const article: Article = {
      id: "123e4567-e89b-12d3-a456-426614174000",
      site_id: "123e4567-e89b-12d3-a456-426614174001",
      title: "Minimal Article",
      slug: "minimal-article",
      language: "en",
      word_count: 0,
      reading_time: 1,
    };
    expect(article.excerpt).toBeUndefined();
    expect(article.content).toBeUndefined();
    expect(article.featured_image_url).toBeUndefined();
    expect(article.author_id).toBeUndefined();
    expect(article.sources).toBeUndefined();
  });

  it("should validate Category type structure", () => {
    const category: Category = {
      id: "123e4567-e89b-12d3-a456-426614174002",
      site_id: "123e4567-e89b-12d3-a456-426614174001",
      name: "Tecnologia",
      slug: "tecnologia",
      description: "Artigos sobre tecnologia",
      sort_order: 1,
    };
    expect(category.name).toBe("Tecnologia");
    expect(category.slug).toBe("tecnologia");
  });
});

describe("formatDate", () => {
  const { formatDate } = require("../lib/api");

  it("should format a valid date string", () => {
    const result = formatDate("2026-07-30T10:00:00Z");
    expect(result).toContain("30");
    expect(result).toContain("julho");
    expect(result).toContain("2026");
  });

  it("should return empty string for undefined", () => {
    expect(formatDate(undefined)).toBe("");
  });

  it("should return empty string for empty string", () => {
    expect(formatDate("")).toBe("");
  });
});

describe("readingTimeLabel", () => {
  const { readingTimeLabel } = require("../lib/api");

  it("should return singular for 1 minute", () => {
    expect(readingTimeLabel(1)).toBe("1 min de leitura");
  });

  it("should return plural for multiple minutes", () => {
    expect(readingTimeLabel(5)).toBe("5 min de leitura");
  });

  it("should handle zero", () => {
    expect(readingTimeLabel(0)).toBe("0 min de leitura");
  });
});
