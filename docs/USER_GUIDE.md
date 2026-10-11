# Author and administrator guide

The publication and studio interfaces are Persian and right-to-left. This guide describes their controls in English. The local studio is <http://localhost:3000/studio>; the intended production address is `https://blog.expertsacademy.ir/studio` once deployed.

## Accounts and permissions

The deployment operator creates the first administrator using the [bootstrap command](../README.md#create-the-first-administrator). The public registration form creates authors only.

| Action                                        | Author | Administrator |
| --------------------------------------------- | ------ | ------------- |
| Create, edit, and publish own active articles | Yes    | Yes           |
| Edit other authors' active articles           | No     | Yes           |
| Upload cover and inline images                | Yes    | Yes           |
| Archive, soft-delete, and restore articles    | No     | Yes           |
| Create categories and tags                    | No     | Yes           |
| Moderate comments                             | No     | Yes           |
| Create, edit, and disable author accounts     | No     | Yes           |

Authors can publish without administrator approval. A disabled account cannot sign in or continue using an existing session. Disabling an account does not hide its published articles.

## Register and sign in

1. Open the studio and select the registration option below the login form.
2. Enter your email, password, and password confirmation. The passwords must match. The form requires a minimum length of 12 and at most 72 UTF-8 bytes; non-ASCII characters can use several bytes. The API enforces a 12–72 byte password range.
3. Both password fields are hidden by default. Use each field's eye button to reveal or hide its value. The login password has the same control.
4. Successful registration signs you in immediately as an author. Public signup has no invitation or email verification step.
5. On later visits, sign in with your email and password. Sessions expire after eight hours.

If the rate-limit message appears, wait at least a minute before trying again. Sign out when using a shared device. There is currently no password reset or password-change screen.

## Studio tabs

Administrators have three separate tabs: **Articles**, **Comment management**, and **Author management**. Authors see the article workspace. Categories and tags are created within the administrator's article workspace.

Switching tabs keeps the current unsaved article and editor state in the page. It does not save them. Refreshing, signing out, closing the page, or opening a different article can lose unsaved changes. Tab controls support arrow keys and Home/End. Controls are temporarily disabled during uploads and saves.

## Create an article

Choose the new-article control and complete the following fields:

| Field                  | Behavior                                                       |
| ---------------------- | -------------------------------------------------------------- |
| Title                  | Required; up to 240 Unicode characters                         |
| Article address (slug) | Unique; 1–200 Unicode letters, digits, or hyphens; no spaces   |
| Excerpt                | Up to 500 characters; used in cards, SEO descriptions, and RSS |
| Content format         | Plain text by default, or HTML                                 |
| Cover image            | Optional PNG/JPEG upload                                       |
| Category               | Optional; selected from administrator-created categories       |
| Tags                   | Up to 20 distinct tags                                         |
| Status                 | Draft or publicly published                                    |

Persian slugs are supported. Keep the address stable after publication: changing it does not create a redirect, and existing links to the old address will need updating.

### Plain text and HTML

Plain text preserves paragraphs and line breaks; HTML-looking text such as `<b>` is displayed literally. Managed image and link controls work in this format too.

HTML mode provides paragraph, heading, bold, quotation, and code buttons. Text areas contain HTML source rather than a visual rich-text canvas. Switching formats does not convert the source: HTML tags become literal text in plain mode and are interpreted again in HTML mode. Review the preview after changing formats.

For example, HTML content can contain:

```html
<h2>Why architecture boundaries matter</h2>
<p>Clear boundaries make business rules easier to change and test.</p>
<blockquote>Dependencies should point toward stable rules.</blockquote>
<ul>
  <li>Define the contract</li>
  <li>Implement the adapter</li>
  <li>Test the behavior</li>
</ul>
```

Scripts, event handlers, iframes, and unsafe URLs are removed from rendered content. Existing articles reopen in the new editor without a database migration. Raw image tags in older HTML remain source text; use the inline upload control when you want dedicated image resizing and alternative-text fields.

### Cover image

Select a PNG or JPEG using the cover upload control. The file must be at most 5 MiB, at most 12 million pixels, and no more than 8,000 pixels on either side. The server validates and re-encodes the image. Save the article after the preview appears.

A cover appears in the article header, publication cards, and sharing metadata. Without one, the site's default artwork is used where appropriate. Removing the cover and saving removes the article's reference; it does not delete the uploaded file. Cover upload does not insert an inline image into the article body.

## Inline images and resizing

1. Click at the desired insertion point in a text block.
2. Use the inline-image button and choose a PNG or JPEG. The same upload limits as the cover apply.
3. Repeat to insert multiple images throughout the article. Each image has independent controls.
4. Set the width using the numeric field or slider, from 80 to 2,000 pixels. The display preserves the aspect ratio and shrinks to fit the available page width. The original stored file is not resized by this control.
5. Add alternative text describing the image, up to 300 characters. Move the block up/down if needed, or remove it from the article.
6. Check the content preview, save, and reopen the article to continue editing.

This workflow works in both plain-text and HTML mode. The document can contain at most 200 total text/image/link blocks, with a combined content size of 500,000 bytes. Removing a block does not delete its file from storage.

Uploaded files are public by URL as soon as the upload succeeds, including images for drafts or articles that have not been saved. Do not upload confidential images.

HTML mode can also reference an externally hosted HTTPS image directly:

```html
<img
  src="https://your-media-host.example/architecture.png"
  alt="Architecture layers"
  width="640"
/>
```

External HTML images depend on that host remaining available. Managed uploads and cover images use the site's own media storage.

## Internal and external links

1. Select the text you want to turn into a link, or place the cursor where a new link should appear.
2. Use the link button and enter the link text.
3. For an internal article, choose the site-article option, type at least two characters, and select a published article from the results.
4. For an external destination, choose the other-address option and enter a complete `http://` or `https://` URL.
5. Insert the link. You can edit the text or destination later in its block, move it, or remove it.

Links are supported in both content formats. The API also permits root-relative paths starting with a single `/`. Script/data URLs, protocol-relative `//` addresses, and URLs containing credentials are rejected by managed-link validation. Complete or cancel the open link panel before saving or starting a new article.

Internal search shows published articles only. Links do not automatically track later slug changes, deletion, or unpublishing; review affected articles when a destination changes.

## Save, publish, and edit

For private content, select **Draft**, then save the draft. Draft articles are excluded from the public API, search, popular list, RSS, and sitemap. Their uploaded image URLs remain public.

To publish, select the public status and use the save-and-publish control. The article must contain text after sanitization; an image-only article cannot be published. Publication time and reading time are calculated automatically. Republishing retains the original first publication date.

Check the success message. There is no autosave, and the preview does not persist changes. Save before leaving the page or opening another article.

Select an article in the list to edit it. Authors can edit their own active articles; administrators can edit everyone's. An administrator's edit preserves the original author. Use the view-article link to open published output.

Saving revalidates the article and public lists through the internal cache webhook. If it fails, periodic ISR remains a fallback; cached content may stay visible while the API/web connection is unhealthy. To unpublish, save as a draft. The public API hides it immediately, but an already cached page or a previously shared image may remain accessible. For urgent removal, contact the operator to follow the [cache procedure](OPERATIONS.md#health-logs-and-cache-recovery).

The article list returns at most 200 recently updated records without pagination. For administrators, the all/mine/archive filters operate within this returned set; they do not guarantee access to older records. Trash is a separate list capped at 200. Pagination beyond these limits is not implemented.

## Archive, delete, and restore

Administrators can archive or delete their own or another author's articles from the article list. Delete requires confirmation in the studio.

- **Archive:** removes the article from public queries and keeps it in the administrator's archive. The author cannot republish it while archived.
- **Delete:** moves it to trash using soft deletion. Content, comments, and tags remain stored.
- **Restore to draft:** restores an archived or deleted article as a private draft and preserves the original author. Edit and publish explicitly to make it public again.

The API also enforces these permissions. An old editor form cannot republish an article after the administrator archives or deletes it. There is no permanent deletion control or endpoint. Public-page cache caveats still apply.

## Categories and tags

In the administrator's article workspace, choose category or tag, enter its display name and unique slug, and create it. For example, use `Software engineering` and `software-engineering`. Persian names and slugs are allowed; use hyphens instead of spaces in slugs. Authors can select existing taxonomy but cannot create it. Rename and delete controls are not implemented.

## Manage author accounts

Open the **Author management** tab as an administrator. Enter the author's name, email, and initial password, then create the account. The account always has the author role, and your administrator session is preserved. The initial-password field is masked; the eye and confirmation controls described above apply to login/public registration. Deliver the initial password through your organization's approved channel.

Use edit beside an account to update its name, email, or active/disabled state. Disabling blocks login and existing sessions on the next authenticated request while preserving the account's articles. Reactivating permits login again. The list is paginated in groups of 20. This form cannot reset passwords, change roles, or edit administrator accounts.

## Submit comments and replies

Below a published article, enter your name, email, and comment. Names must have 2–100 characters and comment text 2–5,000. Signed-in users submit with their account name and email. Email addresses are never displayed in comment responses.

Every submission starts pending, including administrator and author comments. To reply, use the reply control beneath the target comment; its form opens in that branch. Cancel closes it. Conversations can have eight levels including the root; the last level has no reply control.

A new comment queues a notification to the article author. Approval of a reply queues an email to the parent commenter. In development, mock mail sends nothing externally. Authors receive notifications but only administrators can moderate.

## Moderate comments

Open the **Comment management** tab as an administrator and choose pending, approved, or spam. The workspace displays the comment name, text, date, and reply indicator.

- **Approve and publish:** makes an eligible comment available from the public API. Reload the article to see it.
- **Spam:** hides it and removes it from the pending queue.
- **Return to review:** makes it pending again and hides it publicly.
- **Article and reply:** opens the public article so you can reply in the correct branch; the new reply still needs approval.
- **Refresh:** reloads the selected status from the API.

A reply can be approved only when its parent is approved. If needed, locate the parent in the pending or spam status and approve it first. Hiding a parent hides all descendants regardless of their own status.

Each status returns its oldest 200 records with no pagination. Processing pending items removes them from that queue, allowing later pending items to appear after refresh. Approved/spam lists do not page through additional records merely by refreshing; access beyond their first 200 is currently limited.

## Read the publication

Search suggestions appear after at least two characters; Escape closes them. Category and tag links filter articles. The popular list ranks lifetime article views rather than weekly readership or unique visitors. Light/dark preference is stored in the browser. RSS is available at `/rss.xml`.

## Common problems

| Problem                                     | What to check                                                                                  |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Invalid input                               | Length limits, slug syntax, image dimensions, URL scheme, or password confirmation             |
| Duplicate information                       | An email, article slug, or taxonomy name/slug already exists                                   |
| Permission denied                           | The operation is administrator-only or belongs to another author                               |
| Sign-in required                            | The session expired or the account was disabled; preserve unsaved text before signing in again |
| Save control disabled                       | An upload/save is running, or an unfinished link panel needs insertion or cancellation         |
| Image looks narrower than its setting       | The available article/page width limits its displayed size                                     |
| A write is temporarily unavailable          | The operator should inspect Redis, readiness, and API logs                                     |
| Email did not arrive                        | The operator should check mock/SMTP settings, spam folders, and the email outbox               |
| Old content remains after a successful save | The operator should inspect the revalidation webhook and web cache                             |

Password recovery, verified email ownership, invitations, and role changes are not provided by the UI. Account maintenance outside the available controls requires an approved operator process; never store a raw password in PostgreSQL.
