# ExpertsAcademy Blog

وبلاگ سازمانی اکسپرتس آکادمی برای `https://blog.expertsacademy.ir` با Go، PostgreSQL، Redis و Next.js 15.

## اجرای محلی

پیش‌نیاز: Docker Engine و Compose v2، Python 3 و حداقل ۶ گیگابایت فضای آزاد برای اولین build.

```bash
./scripts/init-env.sh
docker compose up --build -d
```

مجله: <http://localhost:3000> — پنل نویسندگان: <http://localhost:3000/studio>

اسکریپت آغازین `.env` را با چهار راز تصادفی مستقل می‌سازد و تنظیمات موجود را حفظ می‌کند؛ برای نسخه‌های قبلی، فقط راز جدید `REVALIDATION_SECRET` را در صورت نبودن اضافه می‌کند. Migration پیش از API اجرا می‌شود. دیتابیس و تصویرهای آپلودشده در volumeهای پایدار جدا نگهداری می‌شوند؛ PostgreSQL و Redis پورت عمومی ندارند. محتوای آزمایشی و حساب پیش‌فرض در دیتابیس ایجاد نمی‌شود.

## ایجاد مدیر اول

```bash
export ADMIN_EMAIL='erfan.najafi8213@gmail.com'
export ADMIN_NAME='Academy Admin'
read -rs -p 'Admin password (12–72 bytes): ' ADMIN_PASSWORD
export ADMIN_PASSWORD
docker compose run --rm -e ADMIN_EMAIL -e ADMIN_NAME -e ADMIN_PASSWORD --entrypoint admin api
unset ADMIN_PASSWORD
```

سپس وارد پنل شوید. نویسندگان تنها با ایمیل و رمز عبور ثبت‌نام می‌کنند؛ ثبت‌نام عمومی نقش مدیر نمی‌سازد. از پنل، دسته و برچسب بسازید، مقاله بنویسید و نظرات را تأیید کنید.

## ساختار

- `apps/api`: Domain، Use Case، Repository مبتنی بر sqlc و HTTP مبتنی بر chi.
- `apps/api/db`: schema، migrationهای golang-migrate و کوئری‌ها.
- `apps/web`: مجله فارسی، SEO، جستجو، نظرات و پنل نویسندگان.
- `docker-compose.yml` و `Caddyfile`: سرویس‌ها، healthcheck و درگاه HTTPS.
- [معماری و قراردادهای API](docs/ARCHITECTURE.md)
- [راهنمای مدیر و نویسنده](docs/USER_GUIDE.md)
- [راهنمای استقرار و نگهداری](docs/OPERATIONS.md)
- [گزارش آزمون‌ها و محدودیت‌های تأیید](docs/VERIFICATION.md)

## آزمون‌ها

```bash
cd apps/api
go test -race ./...
go vet ./...
cd ../web
npm ci
npm run typecheck
npm test
npm run build
```

آزمون یکپارچهٔ Linux با دیتابیس و Redis واقعی: `./scripts/test-integration.sh`. این آزمون یک schema موقت می‌سازد و پاک می‌کند؛ دیتای اصلی را حذف نمی‌کند. متغیرهای `TEST_DATABASE_URL` و `TEST_REDIS_URL` در CI برای همین تست‌ها استفاده می‌شوند. بدون آن‌ها، تست‌های یکپارچه با گزارش skip کنار گذاشته می‌شوند.

## استقرار رسمی

در `.env` آدرس `SITE_URL=https://blog.expertsacademy.ir`، مقدار `CADDY_SITE=blog.expertsacademy.ir`، `APP_ENV=production`، پورت‌های `80/443` و SMTP واقعی را تنظیم کنید. پس از اتصال DNS به سرور و باز بودن پورت‌ها، `docker compose up --build -d` گواهی TLS را از طریق Caddy دریافت می‌کند. پیش از انتشار عمومی، مراحل [OPERATIONS](docs/OPERATIONS.md) را انجام دهید.

## مسیر مهاجرت از دمو

`static/`، `legacy/`، `content-seed/` و `index.html` قبلی به‌عنوان آرشیو حفظ شده‌اند. سرویس جدید از این فایل‌ها اجرا نمی‌شود و محتوای Ghost به‌طور خودکار به بانک سازمانی وارد نمی‌شود. [راهنمای دمو قبلی](docs/LEGACY_DEMO.md) محفوظ است. GitHub Pages اجرای Go، PostgreSQL یا Next.js server را پشتیبانی نمی‌کند؛ DNS رسمی باید به میزبان کانتینری اشاره کند. اسکریپت قدیمی `deploy-demo.sh` مخصوص انتشار آرشیو است و نباید برای نسخهٔ سازمانی استفاده شود.
