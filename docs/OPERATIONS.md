# اجرا، استقرار و نگهداری

## محیط محلی

از ریشه `./scripts/init-env.sh` و `docker compose up --build -d` را اجرا کنید. `.env` در git نادیده گرفته می‌شود و با دسترسی فقط مالک ساخته می‌شود. رازها به‌صورت hex تولید می‌شوند تا URL دیتابیس و Redis بدون escape قابل استفاده باشد. اگر رمز را دستی عوض می‌کنید، کاراکترهای رزروشدهٔ URL باید encode شوند؛ تغییر POSTGRES_PASSWORD در env، رمز دیتابیس موجود را خودکار تغییر نمی‌دهد.

Compose شامل postgres، redis، migrate، api، web و gateway است. migrate پس از healthy شدن postgres اجرا می‌شود؛ API فقط بعد از migration موفق و Redis سالم آغاز می‌شود. frontend و gateway نیز منتظر وابستگی سالم هستند. Migration اجباری در startup خود API اجرا نمی‌شود تا چند replica برای DDL رقابت نکنند.

```bash
docker compose ps -a
docker compose logs --tail=100 api web migrate
docker compose down
```

`down` volumeها را نگه می‌دارد. از `down -v` برای دیتای واقعی استفاده نکنید. در اولین اجرا دیتابیس خالی است؛ حساب مدیر و مقاله باید توسط شما ساخته شوند. محیط توسعهٔ Mock هیچ ایمیلی بیرون نمی‌فرستد.

## استقرار روی دامنه رسمی

۱. یک سرور Linux با Docker/Compose، فضای کافی برای imageها و backup و دسترسی پایدار به SMTP آماده کنید.
۲. رکورد A و در صورت استفاده AAAA دامنه `blog.expertsacademy.ir` را به سرور وصل کنید. گواهی HTTPS تا DNS درست و پورت‌های ۸۰ و ۴۴۳ قابل‌دسترسی نباشند صادر نمی‌شود.
۳. `.env` را از نمونه بسازید و این مقادیر را تنظیم کنید:

```dotenv
APP_ENV=production
SITE_URL=https://blog.expertsacademy.ir
CADDY_SITE=blog.expertsacademy.ir
HTTP_PORT=80
HTTPS_PORT=443
MAIL_DRIVER=smtp
SMTP_PORT=587
```

۴. SMTP_HOST، SMTP_USERNAME، SMTP_PASSWORD و SMTP_FROM را با اطلاعات سرویس سازمان پر کنید. From باید مطابق سیاست سرویس و SPF/DKIM/DMARC دامنهٔ ایمیل باشد. این آداپتور STARTTLS روی 587 را پشتیبانی می‌کند؛ implicit TLS روی 465 پشتیبانی نمی‌شود.
۵. JWT_SECRET را مستقل، تصادفی و حداقل ۳۲ بایت نگه دارید. ثبت‌نام نویسندگان با ایمیل و رمز عبور و بدون دعوت انجام می‌شود.
۶. `docker compose up --build -d` را اجرا کنید. SITE_URL در خروجی client وارد می‌شود؛ تغییر آن نیازمند rebuild وب است.
۷. مدیر اول را مطابق README بسازید، یک مقاله منتشر کنید، نظر مهمان و reply را تأیید و تحویل واقعی هر دو ایمیل را بررسی کنید.
۸. سلامت، backup و هشدار صف ایمیل را به ابزار مانیتورینگ سازمان متصل کنید. این مخزن DNS یا حساب SMTP شما را خودکار تغییر نمی‌دهد.

پورت دیتابیس و Redis در Compose منتشر نمی‌شود. Caddy گواهی را تمدید و HTTP را به HTTPS هدایت می‌کند. شبکهٔ اختصاصی `172.30.58.0/24` برای اعتماد به forwarded client IP تنظیم شده؛ در صورت تداخل subnet، هم Compose و هم TRUSTED_PROXY_CIDRS را همزمان عوض کنید. یک proxy خارجی مانند CDN نیاز به تنظیم دقیق زنجیرهٔ trusted proxies دارد؛ هدر IP کاربر را بدون بررسی اعتماد نکنید.

API بدون root، filesystem فقط‌خواندنی و capabilityهای حذف‌شده اجرا می‌شود. web بدون root است ولی `.next` قابل نوشتن است چون ISR در زمان اجرا cache می‌نویسد. راه‌اندازی مجدد container ممکن است cache را بازسازی کند؛ دادهٔ محتوا در PostgreSQL محفوظ است. در چند replica، cache فعلی Next محلی است؛ تغییرات حداکثر با چرخهٔ revalidate منتشر می‌شوند. برای invalidation فوری و scale گسترده، cache handler مشترک Next را مطابق نیاز استقرار اضافه کنید.

## پشتیبان‌گیری و بازیابی

پشتیبان را روی مسیر امن، سپس در یک محل جدا از سرور نگه دارید. دستور زیر رمز را در خروجی نشان نمی‌دهد:

```bash
umask 077
mkdir -p backups
docker compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > "backups/blog-$(date +%Y%m%d-%H%M%S).dump"
```

بازیابی را ابتدا روی دیتابیس جدا تمرین کنید. پس از تهیهٔ پشتیبان از وضعیت موجود، فقط روی مقصد خالی تأییدشده:

```bash
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --exit-on-error' < backups/selected.dump
```

volume مربوط به `caddy_data` حاوی کلیدهای گواهی است و باید خصوصی بماند. Redis دادهٔ اصلی محتوا نیست؛ از بین رفتن آن باعث از دست رفتن مقاله یا بازدیدهای ذخیره‌شده نمی‌شود. Redis با AOF اجرا می‌شود. اطلاعات ایمیل مهمان در PostgreSQL دادهٔ شخصی است؛ سیاست نگهداری و حذف سازمان را اعمال کنید.

## صف ایمیل

بررسی jobهای متوقف‌شده بدون نمایش آدرس مخاطب:

```bash
docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"' <<'SQL'
SELECT id, attempts, created_at, available_at
FROM email_outbox
WHERE sent_at IS NULL AND attempts >= 12
ORDER BY created_at;
SQL
```

پس از اصلاح SMTP، job مشخص را با `UPDATE email_outbox SET attempts=0, available_at=now() WHERE id='UUID' AND sent_at IS NULL;` دوباره آماده کنید. برای اجرای برنامه‌ریزی‌شدهٔ پاک‌سازی، jobهای sent قدیمی را طبق سیاست سازمان حذف کنید. صف دارای تضمین at-least-once است؛ retry پس از ابهام پاسخ SMTP ممکن است ایمیل تکراری ایجاد کند.

## توسعه بدون Docker برای برنامه‌ها

Go 1.24+ (CI و image: 1.27)، Node 22 و sqlc 1.30.0 لازم‌اند. PostgreSQL و Redis را محلی یا در محیط توسعهٔ مجزا در دسترس قرار دهید. فایل‌های نمونهٔ env را کپی و مقادیر واقعی محلی را تنظیم کنید. Go فایل env را خودکار نمی‌خواند؛ قبل از اجرا متغیرها را export کنید.

```bash
cd apps/api
set -a
source .env
set +a
go run ./cmd/api
```

در ترمینال دیگر:

```bash
cd apps/web
cp .env.example .env.local
npm ci
npm run dev
```

در حالت dev، Next درخواست‌های `/api/v1` را به API_INTERNAL_URL بازنویسی می‌کند. با TRUSTED_PROXY_CIDRS خالی، API IP فرایند Next را می‌بیند و محدودیت نرخ میان کاربران dev مشترک است. در production Caddy مستقیم `/api/v1` را به API می‌فرستد.

## migration و انتشار نسخه

قبل از انتشار، unit/integration tests، typecheck، build و تولید مجدد sqlc باید موفق باشند. migration جدید با شمارهٔ بعدی و SQL up/down افزوده شود؛ فایل اولیهٔ اجراشده روی production را بازنویسی نکنید. schema.sql تصویر نهایی است. قبل از تغییر مخرب، backup تهیه و restore را آزمایش کنید. اجرای دوبارهٔ migrate up در آخرین نسخه بی‌اثر است.

CI سرویس‌های PostgreSQL/Redis جدا ایجاد می‌کند و بعد از تست، down/up migration را فقط روی همان دیتابیس دورریختنی اجرا می‌کند. اسکریپت local integration مخصوص Docker Engine روی Linux است چون از IP خصوصی کانتینر استفاده می‌کند. در Docker Desktop، TEST_DATABASE_URL و TEST_REDIS_URL را روی دیتابیس تست قابل‌دسترسی تنظیم و `go test -race ./...` را اجرا کنید.

## سلامت و انتشار اضطراری

`/healthz` و `/readyz` داخل شبکه روی API در دسترس‌اند. gateway فقط API نسخه‌دار را عمومی می‌کند. اگر readiness شکست خورد، ابتدا `docker compose logs postgres redis api migrate` را بررسی کنید. route، request_id، status و duration در لاگ JSON هستند؛ تنظیم retention و جمع‌آوری آن با سیستم مانیتورینگ استقرار انجام می‌شود.

برای حذف اضطراری نسخهٔ cacheشده پس از draft کردن مقاله، `docker compose up -d --force-recreate web` cache محلی آن container را از نو می‌سازد. این کار را بعد از اطمینان از وضعیت PostgreSQL انجام دهید. در استقرار چندگرهی، تمام replicaها باید refresh شوند.

برای بررسی وابستگی‌ها، `npm audit` و ابزار `govulncheck ./...` را در CI امنیتی سازمان اجرا کنید و releaseهای امنیتی Next 15 را دنبال کنید. نسخه‌ها و lockfileها در مخزن ثبت شده‌اند؛ imageهای پایه در استقرار با حساسیت بالا بهتر است پس از تأیید به digest قفل شوند.

## خطای password authentication failed در migrate

رمز `POSTGRES_PASSWORD` فقط هنگام ساخت volume جدید توسط image PostgreSQL اعمال می‌شود. تغییر `.env` یا بازسازی کانتینر، رمز نقش موجود در دیتابیس را تغییر نمی‌دهد. اگر migrate با این خطا متوقف شد، ابتدا مطمئن شوید تنظیمات فعلی همان رمز موردنظر شماست. برای هماهنگ کردن نقش محلی با رمز فعلی کانتینر، بدون حذف داده‌ها:

```bash
docker compose exec -T postgres sh -c 'exec psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' <<'SQL'
\getenv configured_user POSTGRES_USER
\getenv configured_password POSTGRES_PASSWORD
SELECT format('ALTER ROLE %I PASSWORD %L', :'configured_user', :'configured_password')
\gexec
SQL

docker compose up -d api web gateway
```

این فرمان مخصوص PostgreSQL محلی همین Compose است و رمز نقش دیتابیس را تغییر می‌دهد؛ اگر مصرف‌کنندهٔ دیگری به همان دیتابیس متصل است، تنظیمات آن نیز باید هماهنگ باشد. رمز در خروجی ترمینال نمایش داده نمی‌شود. healthcheck اکنون با اتصال TCP احرازهویت‌شده تست می‌کند تا اختلاف رمز به‌اشتباه healthy گزارش نشود. برای رفع این خطا نیازی به `down -v`، حذف volume یا force کردن شمارهٔ migration نیست.

## رسانه و کش انتشار

قبل از build نسخهٔ جدید، `bash scripts/init-env.sh` را اجرا کنید؛ در `.env` موجود فقط REVALIDATION_SECRET جدید اضافه می‌شود و رمز PostgreSQL/Redis عوض نمی‌شود. این راز در محیط runtime API و web یکسان و خصوصی است. در اجرای جداگانهٔ Go/Next، UPLOAD_DIR، WEB_REVALIDATION_URL و REVALIDATION_SECRET را مطابق env.example هر برنامه تنظیم کنید.

volume `uploads_data` باید کنار backup PostgreSQL پشتیبان‌گیری و بازیابی شود؛ صرفاً dump دیتابیس شامل فایل‌های عکس نیست. مالک مسیر آپلود در image API کاربر 10001 است. فایل‌ها عمومی و UUID دارند؛ رسانهٔ محرمانه آپلود نکنید. برای چند replica روی میزبان‌های جدا، MediaStore باید با storage مشترک پیاده‌سازی شود؛ volume محلی Docker میان میزبان‌ها مشترک نیست.

اگر انتشار موفق است اما محتوا تازه نمی‌شود، هشدار `site cache revalidation failed` در لاگ API، دسترسی API به WEB_REVALIDATION_URL و برابر بودن REVALIDATION_SECRET در دو سرویس را بررسی کنید. وب‌هوک بدون توکن معتبر 401 می‌دهد.
