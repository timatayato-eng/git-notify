# git-notify

Discord Bot ที่ส่ง **DM แจ้งเตือนหาคนในทีมโดยตรง** เมื่อมีการ assign งานหรืออัปเดต Issue/Pull request บน GitHub

- แต่ละคนพิมพ์ `/git-notify github:<ชื่อ GitHub>` ใน Discord **ครั้งเดียว** เพื่อผูกบัญชี
- เมื่อมีคนถูก **assign** ใน Issue หรือ PR คนที่ถูก assign จะได้รับ DM
- เมื่อ Issue หรือ PR ถูก **ปิด, merge, เปิดใหม่ หรือเปลี่ยน label** ทุกคนที่ถูก assign อยู่จะได้รับ DM
- คนที่เป็นคนทำการเปลี่ยนแปลงเองจะไม่ได้รับ DM ของตัวเอง

ตัวอย่าง DM:

```text
**[acme/demo] Issue #7** Fix login
alice assign งานนี้ให้คุณ
https://github.com/acme/demo/issues/7
```

## ขั้นตอนการติดตั้ง (แอดมินทำครั้งเดียว)

### 1. สร้าง Discord Bot

1. เข้า [Discord Developer Portal](https://discord.com/developers/applications) แล้วกด **New Application** ตั้งชื่อ เช่น `git-notify`
2. ไปที่เมนู **Bot** → กด **Reset Token** → **Copy** เก็บไว้ใช้เป็น `DISCORD_TOKEN` (ห้ามเผยแพร่ ถ้าหลุดให้กด Reset ใหม่)
3. ไปที่เมนู **OAuth2** → **URL Generator**
   - **Scopes**: ติ๊ก `bot` และ `applications.commands`
   - **Bot Permissions**: ติ๊ก `Send Messages`
4. คัดลอก URL ที่ได้ด้านล่าง เปิดในเบราว์เซอร์ แล้วเลือก server ที่จะเพิ่ม bot (ต้องมีสิทธิ์ **Manage Server**)

**Server ID** (สำหรับ `DISCORD_GUILD_ID`, ไม่บังคับแต่แนะนำ): ใน Discord ไปที่ **User Settings** → **Advanced** → เปิด **Developer Mode** แล้วคลิกขวาที่ไอคอน server → **Copy Server ID**

### 2. ตั้งค่า `.env`

```bash
cp .env.default .env
```

| ตัวแปร | จำเป็น | ความหมาย |
| --- | --- | --- |
| `DISCORD_TOKEN` | ใช่ | Bot token จากขั้นตอนที่ 1 |
| `DISCORD_GUILD_ID` | แนะนำ | Server ID ถ้าใส่ คำสั่ง `/git-notify` จะขึ้นใน server นั้นทันที ถ้าไม่ใส่ คำสั่งจะใช้ได้ทุก server ที่มี bot แต่อาจใช้เวลาสักพักกว่าจะขึ้น |
| `GITHUB_WEBHOOK_SECRET` | แนะนำ | รหัสลับที่ตั้งเอง ใช้ตรวจว่า request มาจาก GitHub จริง สร้างด้วย `openssl rand -hex 32` และต้องใส่ค่าเดียวกันใน GitHub (ขั้นตอนที่ 4) |
| `GITHUB_NOTIFY_ACTIONS` | ไม่ | action ที่จะส่ง DM คั่นด้วยจุลภาค (ดูหัวข้อ "เลือกว่าจะแจ้งเตือนอะไร") |
| `WEBHOOK_PORT` | ไม่ | port ของ server (ค่าเริ่มต้น `8080`) |
| `USER_STORE_PATH` | ไม่ | ไฟล์เก็บการผูกบัญชี (ค่าเริ่มต้น `data/users.json`) |

### 3. รัน server

```bash
docker compose up -d --build
docker compose logs -f
```

หรือรันตรงด้วย Go 1.25 ขึ้นไป: `go run .`

ถ้าเริ่มสำเร็จ จะเห็น log `ลงทะเบียนคำสั่ง /git-notify แล้ว` และ bot จะขึ้นสถานะออนไลน์ใน Discord

การผูกบัญชีของทุกคนเก็บอยู่ในโฟลเดอร์ `data/` ซึ่ง mount ออกมานอก container ข้อมูลจึงไม่หายเมื่อ restart หรือ build ใหม่

server ต้องรับ request จากอินเทอร์เน็ตที่ `http://<host>:8080/webhook/github` ได้ ถ้าทดสอบบนเครื่องตัวเอง ใช้ [ngrok](https://ngrok.com/) (`ngrok http 8080`) หรือ Cloudflare Tunnel ได้

### 4. ตั้ง Webhook ใน GitHub

ไปที่ repo → **Settings** → **Webhooks** → **Add webhook** (ถ้าต้องการทุก repo ในองค์กร ให้ตั้งที่ **Organization Settings** → **Webhooks**)

- **Payload URL**: `https://<host>/webhook/github`
- **Content type**: `application/json` (**ต้องเลือกค่านี้**)
- **Secret**: ค่าเดียวกับ `GITHUB_WEBHOOK_SECRET`
- **Which events**: เลือก **Let me select individual events** แล้วติ๊ก **Issues** และ/หรือ **Pull requests**

กด **Add webhook** แล้วดูแท็บ **Recent Deliveries** ถ้า `ping` ได้ `200` แสดงว่าเชื่อมต่อสำเร็จ

## การใช้งานสำหรับคนในทีม

พิมพ์ใน channel ใดก็ได้ของ Discord server ที่มี bot:

```text
/git-notify github:octocat
```

(ใส่ชื่อ GitHub ของตัวเอง หรือวางลิงก์โปรไฟล์ `https://github.com/octocat` ก็ได้) bot จะตอบกลับเป็นข้อความที่เห็นแค่คนพิมพ์ ถ้าเปลี่ยนชื่อ GitHub ให้พิมพ์คำสั่งใหม่ ข้อมูลเดิมจะถูกแทนที่

คนที่จะรับ DM ได้ต้อง:

- อยู่ใน Discord server เดียวกับ bot
- เปิดรับ DM จากสมาชิกใน server (คลิกชื่อ server → **Privacy Settings** → เปิด **Direct Messages**) ซึ่งปกติเปิดไว้อยู่แล้ว

## เลือกว่าจะแจ้งเตือนอะไร

แบ่งเป็นสองชั้น:

1. **ใน GitHub Webhook** เลือกประเภท event ที่จะส่งมา: **Issues**, **Pull requests** หรือทั้งสองอย่าง event ประเภทอื่นที่ส่งมาจะถูกข้าม
2. **ใน `GITHUB_NOTIFY_ACTIONS`** เลือกว่า action ไหนจะส่ง DM ค่าเริ่มต้นคือ `assigned,closed,reopened,labeled,unlabeled`

action ที่ใช้บ่อย:

| action | เกิดเมื่อ | ใครได้รับ DM |
| --- | --- | --- |
| `assigned` | มีคนถูก assign | คนที่ถูก assign |
| `unassigned` | มีคนถูกเอาออกจากงาน | คนที่ถูกเอาออก |
| `closed` | ปิด Issue หรือ merge/ปิด PR | ทุกคนที่ถูก assign |
| `reopened` | เปิดใหม่อีกครั้ง | ทุกคนที่ถูก assign |
| `labeled` / `unlabeled` | เพิ่ม/ลบ label (เช่นใช้ label เป็นสถานะ `in-progress`, `review`) | ทุกคนที่ถูก assign |
| `edited` | แก้ชื่อหรือรายละเอียด | ทุกคนที่ถูก assign |
| `review_requested` | ขอ review PR | ทุกคนที่ถูก assign PR |

รายการ action ทั้งหมดดูได้ที่ [GitHub webhook events: issues](https://docs.github.com/en/webhooks/webhook-events-and-payloads#issues) และ [pull_request](https://docs.github.com/en/webhooks/webhook-events-and-payloads#pull_request)

ยังไม่รองรับคอลัมน์ Status ใน **GitHub Projects** (เช่น Todo → In Progress → Done) และคอมเมนต์ใน Issue

## Deploy อัตโนมัติด้วย GitHub Actions

`.github/workflows/deploy.yml` จะรัน `go vet` และ `go test` ทุกครั้งที่ push เข้า `main` หรือเปิด PR และจะ deploy เมื่อ push tag รูปแบบ `v1.2.3`

สิ่งที่ต้องตั้งค่าก่อน:

1. **Self-hosted runner** บน server: repo → **Settings** → **Actions** → **Runners** → **New self-hosted runner** ทำตามคำสั่งที่แสดง และเพิ่ม label `prod` เครื่องต้องมี Docker และ Docker Compose และ user ที่รัน runner ต้องใช้คำสั่ง `docker` ได้
2. **Secrets** (repo → **Settings** → **Secrets and variables** → **Actions** → แท็บ **Secrets**):
   - `DISCORD_TOKEN`
   - `GH_WEBHOOK_SECRET` ค่าเดียวกับ Secret ใน GitHub Webhook (GitHub ไม่ให้ตั้งชื่อ secret ขึ้นต้นด้วย `GITHUB_`)
3. **Variables** (แท็บ **Variables**, ไม่บังคับ):
   - `DISCORD_GUILD_ID`
   - `GITHUB_NOTIFY_ACTIONS`
   - `DATA_DIR` โฟลเดอร์บน server ที่เก็บข้อมูลการผูกบัญชี (ค่าเริ่มต้น `/var/lib/git-notify`) ต้องเป็น path ถาวร ไม่ใช่ในโฟลเดอร์ของ runner เพราะ runner จะล้างโฟลเดอร์นั้นทุกครั้งที่ deploy

deploy ด้วย:

```bash
git tag v1.0.0
git push origin v1.0.0
```

## พัฒนาและทดสอบ

```bash
go test ./...
```

- `main.go` โหลด `.env`, เชื่อมต่อ Discord, ลงทะเบียนคำสั่ง และเริ่ม HTTP server
- `core/discord.go` สร้าง Discord session, ลงทะเบียน `/git-notify` และส่ง DM
- `core/http.go` ลงทะเบียน route `/webhook/github`
- `app/discord_command.go` จัดการคำสั่ง `/git-notify`
- `app/user_store.go` เก็บการผูกบัญชี GitHub กับ Discord ลงไฟล์ JSON
- `app/github_webhook.go` ตรวจ signature, เลือกคนที่จะได้ DM และสร้างข้อความ
- `types/github/payload.go` โครงสร้าง payload ของ GitHub ที่ใช้

## แก้ปัญหาเบื้องต้น

- **ไม่เห็นคำสั่ง `/git-notify` ใน Discord**: ตอนเชิญ bot ต้องติ๊ก scope `applications.commands` ถ้าไม่ได้ใส่ `DISCORD_GUILD_ID` อาจต้องรอสักพัก หรือลองปิดเปิด Discord ใหม่
- **GitHub ได้ `401`**: Secret ใน GitHub ไม่ตรงกับ `GITHUB_WEBHOOK_SECRET`
- **GitHub ได้ `400 Invalid JSON`**: Content type ไม่ได้ตั้งเป็น `application/json`
- **GitHub ได้ `200` แต่ไม่มี DM**: ดู log ด้วย `docker compose logs -f`
  - `ยังไม่ได้ผูกบัญชีด้วย /git-notify`: คนนั้นยังไม่ได้พิมพ์คำสั่ง หรือพิมพ์ชื่อ GitHub ผิด
  - `ข้าม event=... action=...`: action นั้นไม่ได้อยู่ใน `GITHUB_NOTIFY_ACTIONS`
  - `ส่ง DM ... ไม่สำเร็จ`: คนนั้นปิดรับ DM หรือไม่ได้อยู่ใน server เดียวกับ bot
- **ไม่มีใครได้ DM ตอนปิด Issue**: Issue นั้นไม่มีคนถูก assign หรือคนที่ถูก assign เป็นคนปิดเอง
