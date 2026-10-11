export interface Taxon {
  id: string;
  name: string;
  slug: string;
}
export interface Post {
  id: string;
  author_id: string;
  author_name?: string;
  category_id: string | null;
  slug: string;
  title: string;
  content_html: string;
  content_json: unknown;
  cover_image_url: string | null;
  excerpt: string;
  reading_time_minutes: number;
  views_count: number;
  status: "draft" | "published" | "archived" | "deleted";
  published_at: string | null;
  created_at: string;
  updated_at: string;
  tags: Taxon[];
}
export interface PostPage {
  items: Post[];
  total: number;
  page: number;
  limit: number;
}
export interface Comment {
  id: string;
  post_id: string;
  parent_id: string | null;
  author_name: string;
  content: string;
  status: string;
  created_at: string;
  replies: Comment[];
}
export interface User {
  id: string;
  name: string;
  email: string;
  role: "admin" | "author" | "reader";
  disabled_at: string | null;
}
export interface AuthorPage {
  items: User[];
  total: number;
  page: number;
  limit: number;
}
export interface Taxonomy {
  categories: Taxon[];
  tags: Taxon[];
}
