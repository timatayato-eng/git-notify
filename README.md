# git-notify

รับ Webhook จาก GitHub แล้วส่งแจ้งเตือนเข้า Discord channel ผ่าน Discord Webhook

- **Push**: แจ้งว่าใคร push กี่ commit ไปที่ branch ไหน พร้อมลิงก์ compare
- **Pull request / Issue**: แจ้งเมื่อเปิด ปิด merge แก้ไข ฯลฯ
- **Assign**: เมื่อมีคนถูก assign ใน PR หรือ Issue จะ mention (`@ชื่อ`) คนนั้นใน Discord ถ้าตั้งค่า `DISCORD_USER_MAP` ไว้
- event อื่นที่เลือกไว้ใน GitHub จะแสดงเป็นข้อความสั้น ๆ พร้อมชื่อ event

## ขั้นตอนการใช้งาน

### 1. สร้าง Discord Webhook

1. ใน Discord คลิกที่ channel ที่ต้องการรับแจ้งเตือน → **Edit Channel** → **Integrations** → **Webhooks**
2. กด **New Webhook** แล้วกด **Copy Webhook URL**
3. เก็บ URL นี้ไว้เป็นความลับ ใครมี URL นี้ก็ส่งข้อความเข้า channel ได้ (ถ้าหลุด ให้ลบ webhook แล้วสร้างใหม่)

### 2. ตั้งค่า `.env`

```bash
cp .env.default .env
```

| ตัวแปร | จำเป็น | ความหมาย |
| --- | --- | --- |
| `DISCORD_WEBHOOK_URL` | ใช่ | URL จากขั้นตอนที่ 1 |
| `GITHUB_WEBHOOK_SECRET` | แนะนำ | ข้อความลับที่ใช้ตรวจว่า request มาจาก GitHub จริง ต้องตรงกับ Secret ใน GitHub (ขั้นตอนที่ 4) |
| `DISCORD_USER_MAP` | ไม่ | จับคู่ GitHub username กับ Discord user ID เช่น `octocat:123456789012345678,alice:987654321098765432` |
| `DISCORD_WEBHOOK_USERNAME` | ไม่ | ชื่อผู้ส่งที่แสดงใน Discord (ค่าเริ่มต้นคือชื่อที่ตั้งไว้ใน webhook) |
| `WEBHOOK_PORT` | ไม่ | port ของ server (ค่าเริ่มต้น `8080`) |

สร้าง secret แบบสุ่มได้ด้วยคำสั่ง `openssl rand -hex 32`

**วิธีหา Discord user ID:** ใน Discord ไปที่ **User Settings** → **Advanced** → เปิด **Developer Mode** แล้วคลิกขวาที่ชื่อคน → **Copy User ID**

### 3. รัน server

ด้วย Docker:

```bash
docker compose up -d --build
docker compose logs -f
```

หรือรันตรงด้วย Go (ต้องใช้ Go 1.25 ขึ้นไป):

```bash
go run .
```

server จะรอรับ request ที่ `http://<host>:8080/webhook/github`

GitHub ต้องเข้าถึง URL นี้จากอินเทอร์เน็ตได้ ถ้าทดสอบบนเครื่องตัวเอง ให้เปิดทางด้วยเครื่องมืออย่าง [ngrok](https://ngrok.com/) (`ngrok http 8080`) หรือ Cloudflare Tunnel แล้วใช้ URL ที่ได้ในขั้นตอนถัดไป

### 4. ตั้ง Webhook ใน GitHub

ไปที่ repo บน GitHub → **Settings** → **Webhooks** → **Add webhook** (ถ้าต้องการให้ทุก repo ในองค์กรแจ้งเตือน ให้ตั้งที่ **Organization Settings** → **Webhooks** แทน)

- **Payload URL**: `https://<host>/webhook/github`
- **Content type**: `application/json` (**ต้องเลือกค่านี้** ถ้าเลือกแบบ form โปรแกรมจะอ่าน payload ไม่ได้)
- **Secret**: ค่าเดียวกับ `GITHUB_WEBHOOK_SECRET`
- **Which events**: เลือก **Let me select individual events** แล้วติ๊ก **Pushes**, **Pull requests**, **Issues** (event ที่ไม่ได้เลือกจะไม่ถูกส่งมา)

กด **Add webhook** แล้ว GitHub จะส่ง `ping` มาทดสอบ ถ้าในแท็บ **Recent Deliveries** ได้ response `200` แสดงว่าเชื่อมต่อสำเร็จ ลอง push หรือเปิด issue แล้วดูข้อความใน Discord ได้เลย

## Deploy อัตโนมัติด้วย GitHub Actions

ไฟล์ `.github/workflows/deploy.yml` จะ

- รัน `go vet` และ `go test` ทุกครั้งที่ push เข้า `main` หรือเปิด pull request
- deploy เมื่อ push tag รูปแบบ `v1.2.3` หรือ `1.2.3` โดยสร้าง `.env` จาก secrets แล้วรัน `docker compose up -d --build` บน server

สิ่งที่ต้องตั้งค่าก่อน:

1. **Self-hosted runner** บน server ที่จะรัน: ไปที่ repo → **Settings** → **Actions** → **Runners** → **New self-hosted runner** ทำตามคำสั่งที่ GitHub แสดง และตอนตั้งค่าให้เพิ่ม label `prod` เครื่องนั้นต้องติดตั้ง Docker และ Docker Compose ไว้ และ user ที่รัน runner ต้องใช้คำสั่ง `docker` ได้
2. **Secrets** (repo → **Settings** → **Secrets and variables** → **Actions** → แท็บ **Secrets**):
   - `DISCORD_WEBHOOK_URL`
   - `GH_WEBHOOK_SECRET` (GitHub ไม่อนุญาตให้ชื่อ secret ขึ้นต้นด้วย `GITHUB_` จึงต้องใช้ชื่อนี้ ค่าคือ secret เดียวกับที่ใส่ใน Webhook)
3. **Variables** (แท็บ **Variables**, ไม่บังคับ): `DISCORD_USER_MAP`, `DISCORD_WEBHOOK_USERNAME`

จากนั้น deploy ด้วย:

```bash
git tag v1.0.0
git push origin v1.0.0
```

## พัฒนาและทดสอบ

```bash
go test ./...
```

โครงสร้างโปรเจค:

- `main.go` โหลด `.env` และเริ่ม server
- `core/discord.go` ส่งข้อความไป Discord Webhook (ping เฉพาะ user ที่ระบุ ไม่ ping `@everyone`)
- `core/http.go` ลงทะเบียน route `/webhook/github`
- `app/github_webhook.go` ตรวจ signature, แปลง event เป็นข้อความ และจัดการการ assign
- `types/github/payload.go` โครงสร้าง payload ของ GitHub ที่ใช้

## แก้ปัญหาเบื้องต้น

- **GitHub ได้ `401`**: Secret ใน GitHub ไม่ตรงกับ `GITHUB_WEBHOOK_SECRET`
- **GitHub ได้ `400 Invalid JSON`**: Content type ใน GitHub ไม่ได้ตั้งเป็น `application/json`
- **GitHub ขึ้น timeout หรือเชื่อมต่อไม่ได้**: server เข้าจากอินเทอร์เน็ตไม่ได้ ให้ตรวจ firewall, port หรือ tunnel
- **ได้ `200` แต่ไม่มีข้อความใน Discord**: ดู log ด้วย `docker compose logs -f` ถ้าเจอ `ไม่สามารถส่ง Discord notify` แสดงว่า `DISCORD_WEBHOOK_URL` ผิดหรือ webhook ถูกลบไปแล้ว
- **ไม่มีการ mention ตอน assign**: ยังไม่ได้ใส่ GitHub username คนนั้นใน `DISCORD_USER_MAP`
