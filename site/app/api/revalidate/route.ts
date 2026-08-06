import { revalidatePath, revalidateTag } from "next/cache";
import { NextRequest, NextResponse } from "next/server";

// ISR revalidation webhook — called by the Nexora API after every publish
// (POST {site}/api/revalidate, header x-revalidate-token). Revalidates the
// homepage, the article page and every cached page tagged "articles".
export async function POST(request: NextRequest) {
  const token = request.headers.get("x-revalidate-token");
  const expected = process.env.SITE_REVALIDATE_TOKEN;

  if (!expected || token !== expected) {
    return NextResponse.json({ error: "invalid token" }, { status: 401 });
  }

  let slug = "";
  try {
    const body = await request.json();
    if (typeof body?.slug === "string") {
      slug = body.slug;
    }
  } catch {
    return NextResponse.json({ error: "invalid body" }, { status: 400 });
  }

  if (!slug) {
    return NextResponse.json({ error: "slug is required" }, { status: 400 });
  }

  revalidatePath("/");
  revalidatePath(`/${slug}`);
  revalidateTag("articles");

  return NextResponse.json({ revalidated: true, slug });
}
