# git-notify

Discord Bot ที่ส่ง **DM แจ้งเตือนหาคนในทีมโดยตรง** เมื่อมีการ assign งาน หรืออัปเดต Issue / Pull request บน GitHub

ตัวอย่าง DM ที่ได้รับ:

```text
[acme/demo] Issue #7 Fix login
alice assign งานนี้ให้คุณ
https://github.com/acme/demo/issues/7
```

## ภาพรวมการทำงาน

```text
GitHub (มีคน assign / ปิด issue / เปลี่ยน label)
   │  ส่ง Webhook
   ▼
git-notify (โปรแกรมนี้ รันบน server)
   │  ดูว่า GitHub user นี้คือใครใน Discord
   ▼
Discord Bot ส่ง DM หาคนนั้น
```

แต่ละคนในทีมผูกบัญชี GitHub กับ Discord ของตัวเองครั้งเดียว ด้วยการพิมพ์ใน Discord ว่า

```text
/git-notify github:<ชื่อ GitHub ของตัวเอง>
```

## สิ่งที่ต้องเตรียม

- บัญชี Discord ที่มีสิทธิ์ **Manage Server** ใน server ของทีม (ใช้ตอนเชิญ bot)
- สิทธิ์ **Admin** ของ repo บน GitHub (ใช้ตอนตั้ง Webhook)
- เครื่องหรือ server ที่จะรันโปรแกรม ติดตั้ง [Docker](https://docs.docker.com/get-docker/) หรือ [Go 1.25+](https://go.dev/dl/) ไว้
- URL ที่ GitHub เข้าถึงได้จากอินเทอร์เน็ต ถ้าทดสอบบนเครื่องตัวเอง ใช้ [ngrok](https://ngrok.com/) ได้

ทำตามลำดับ: **ส่วนที่ 1 Discord** → **ส่วนที่ 2 รันโปรแกรม** → **ส่วนที่ 3 GitHub Webhook** → **ส่วนที่ 4 ให้ทีมผูกบัญชี**

---

## ส่วนที่ 1: ตั้งค่า Discord

### 1.1 สร้าง Discord Application

1. เข้า [Discord Developer Portal](https://discord.com/developers/applications) แล้วล็อกอิน
2. กด **New Application** มุมขวาบน
3. ตั้งชื่อ เช่น `git-notify` ติ๊กยอมรับเงื่อนไข แล้วกด **Create**

ชื่อนี้จะเป็นชื่อของ bot ที่ทุกคนเห็นใน Discord ใส่รูปได้ที่หน้า **General Information**

### 1.2 เอา Bot Token → `DISCORD_TOKEN`

**คืออะไร:** รหัสที่โปรแกรมใช้ล็อกอินเป็น bot ใครมี token นี้ก็ควบคุม bot ได้ ต้องเก็บเป็นความลับ

**เอามาจากไหน:**

1. ในหน้า app เมนูซ้าย กด **Bot**

   ```text
   Overview
     General Information
     Installation
     OAuth2
     Bot          ← กดตรงนี้
   ```

2. ที่หัวข้อ **Token** กด **Reset Token** → **Yes, do it!** (ถ้าเปิด 2FA ไว้ จะให้ใส่รหัส)
3. กด **Copy** แล้วเก็บไว้ใส่ใน `.env` ในส่วนที่ 2

> **ระวังสับสน:** **Application ID**, **Public Key** (หน้า General Information) และ **Client Secret** (หน้า OAuth2) **ไม่ใช่** Bot Token
>
> Token แสดงแค่ครั้งเดียว ถ้าลืม copy ให้กด Reset Token ใหม่ ถ้า token หลุด ให้กด Reset ทันที token เก่าจะใช้ไม่ได้

ในหน้า Bot ไม่ต้องเปิด **Privileged Gateway Intents** ใด ๆ

### 1.3 เชิญ Bot เข้า Server

ต้องทำขั้นตอนนี้ ไม่อย่างนั้น bot จะรับคำสั่งและส่ง DM ไม่ได้

1. เมนูซ้าย กด **OAuth2** แล้วเลื่อนลงไปที่ **OAuth2 URL Generator** (ช่อง Client ID, Client Secret และ Redirects ด้านบนไม่ต้องใช้)
2. **Scopes:** ติ๊ก 2 ช่อง
   - `bot`
   - `applications.commands` (อยู่ด้านล่างของรายการ ถ้าไม่ติ๊ก คำสั่ง `/git-notify` จะไม่ขึ้น)
3. **Bot Permissions:** ติ๊ก **Send Messages**
4. **Integration Type** (ถ้ามี): เลือก **Guild Install**
5. กด **Copy** ที่ **Generated URL** ด้านล่างสุด
6. วาง URL ในเบราว์เซอร์ เลือก server ของทีม แล้วกด **Authorize**

ตรวจสอบ: ในรายชื่อสมาชิกของ server จะมี bot ชื่อ `git-notify` อยู่ ตอนนี้ยังออฟไลน์ จะออนไลน์เมื่อรันโปรแกรมในส่วนที่ 2 (ตัวเลข **Install Count** ในหน้า General Information อัปเดตวันละครั้ง ยังเป็น 0 ได้)

### 1.4 เอา Server ID → `DISCORD_GUILD_ID`

**คืออะไร:** เลข ID ของ Discord server ของทีม ใช้ลงทะเบียนคำสั่ง `/git-notify` ให้ขึ้นใน server นี้ทันที ถ้าไม่ใส่ คำสั่งจะใช้ได้ทุก server ที่มี bot แต่อาจต้องรอสักพักกว่าจะขึ้น

**เอามาจากไหน:**

1. ใน Discord ไปที่ **User Settings** (ไอคอนเฟืองข้างชื่อตัวเองมุมซ้ายล่าง) → **Advanced** → เปิด **Developer Mode**
2. คลิกขวาที่ไอคอน server ในแถบซ้าย → **Copy Server ID**

ได้เลขยาว 17–19 หลัก เช่น `123456789012345678`

---

## ส่วนที่ 2: ตั้งค่าและรันโปรแกรม

### 2.1 สร้างไฟล์ `.env`

```bash
git clone <URL ของ repo นี้>
cd git-notify
cp .env.default .env
```

เปิด `.env` แล้วใส่ค่า:

| ตัวแปร | จำเป็น | คืออะไร | เอามาจากไหน |
| --- | --- | --- | --- |
| `DISCORD_TOKEN` | ใช่ | รหัสล็อกอินของ bot | ข้อ 1.2 |
| `DISCORD_GUILD_ID` | แนะนำ | ID ของ Discord server | ข้อ 1.4 |
| `GITHUB_WEBHOOK_SECRET` | แนะนำ | รหัสลับที่ใช้ตรวจว่า request มาจาก GitHub จริง | **ตั้งขึ้นเอง** ดูด้านล่าง |
| `GITHUB_NOTIFY_ACTIONS` | ไม่ | action ที่จะส่ง DM | ดูหัวข้อ [เลือกว่าจะแจ้งเตือนอะไร](#เลือกว่าจะแจ้งเตือนอะไร) ค่าเริ่มต้นใช้ได้เลย |
| `WEBHOOK_PORT` | ไม่ | port ที่โปรแกรมรอรับ webhook | ค่าเริ่มต้น `8080` |
| `USER_STORE_PATH` | ไม่ | ไฟล์ที่เก็บการผูกบัญชีของทุกคน | ค่าเริ่มต้น `data/users.json` |

**สร้าง `GITHUB_WEBHOOK_SECRET`:** ไม่ได้เอามาจากที่ไหน ให้สุ่มขึ้นมาเองด้วยคำสั่ง

```bash
openssl rand -hex 32
```

นำค่าที่ได้ไปใส่ **2 ที่ให้ตรงกัน**: ใน `.env` และในช่อง **Secret** ของ GitHub Webhook (ส่วนที่ 3)

ตัวอย่าง `.env` ที่กรอกแล้ว (ไม่ต้องใส่เครื่องหมายคำพูด และไม่เว้นวรรครอบ `=`):

```env
DISCORD_TOKEN=MTU2MTk5NTM0ODA3NDAzNzI2MA.Gxxxxx.xxxxxxxxxxxxxxxx
DISCORD_GUILD_ID=123456789012345678
WEBHOOK_PORT=8080
GITHUB_WEBHOOK_SECRET=3f9a1c0d...e7b2
GITHUB_NOTIFY_ACTIONS=assigned,closed,reopened,labeled,unlabeled
USER_STORE_PATH=data/users.json
```

`.env` อยู่ใน `.gitignore` แล้ว ห้าม commit ไฟล์นี้ขึ้น GitHub

### 2.2 รันโปรแกรม

ด้วย Docker (แนะนำสำหรับใช้งานจริง):

```bash
docker compose up -d --build
docker compose logs -f
```

หรือรันตรงด้วย Go (สะดวกตอนทดสอบ):

```bash
go run .
```

ถ้าสำเร็จจะเห็น log:

```text
ลงทะเบียนคำสั่ง /git-notify แล้ว
GitHub webhook server กำลังทำงานที่ port 8080
git-notify กำลังทำงาน... กด CTRL+C เพื่อหยุด
```

และ bot จะขึ้นสถานะออนไลน์ใน Discord

### 2.3 เปิดให้ GitHub เข้าถึงได้

GitHub ต้องส่ง request มาที่ `/webhook/github` ของโปรแกรมได้จากอินเทอร์เน็ต ใช้ `localhost` ไม่ได้

- **บน server ที่มี IP หรือโดเมน:** URL คือ `https://<โดเมน>/webhook/github` หรือ `http://<IP>:8080/webhook/github` (เปิด port 8080 ใน firewall ด้วย)
- **ทดสอบบนเครื่องตัวเอง:** เปิด Terminal อีกหน้าต่างแล้วรัน

  ```bash
  ngrok http 8080
  ```

  นำ URL ในบรรทัด **Forwarding** มาต่อท้ายด้วย `/webhook/github` เช่น `https://skipper-yesterday-heat.ngrok-free.dev/webhook/github`

  ngrok แบบฟรีจะได้ URL ใหม่ทุกครั้งที่เปิดใหม่ ต้องกลับไปแก้ Payload URL ใน GitHub ทุกครั้ง

---

## ส่วนที่ 3: ตั้งค่า GitHub Webhook

ให้โปรแกรมรันอยู่ก่อน (ส่วนที่ 2) ไม่อย่างนั้นการทดสอบ `ping` จะไม่ผ่าน

1. ไปที่ repo บน GitHub → **Settings** → **Webhooks** → **Add webhook**
   (ถ้าต้องการให้ทุก repo ในองค์กรแจ้งเตือน ให้ตั้งที่ **Organization** → **Settings** → **Webhooks** แทน)
2. กรอกแต่ละช่อง:

| ช่อง | ใส่อะไร | หมายเหตุ |
| --- | --- | --- |
| **Payload URL** | URL จากข้อ 2.3 ต่อท้ายด้วย `/webhook/github` | เช่น `https://example.com/webhook/github` |
| **Content type** | `application/json` | **ต้องเปลี่ยน** เพราะค่าเริ่มต้นเป็น `application/x-www-form-urlencoded` ซึ่งโปรแกรมอ่านไม่ได้ |
| **Secret** | ค่าเดียวกับ `GITHUB_WEBHOOK_SECRET` ใน `.env` | ระวังช่องว่างหรือขึ้นบรรทัดใหม่ติดมาตอน copy |
| **SSL verification** | **Enable SSL verification** | ใช้ค่าเริ่มต้น |
| **Which events would you like to trigger this webhook?** | **Let me select individual events** | ดูด้านล่าง |
| **Active** | ติ๊ก | |

3. หลังเลือก **Let me select individual events** จะมี checkbox ขึ้นมา:
   - ติ๊ก **Issues**
   - ติ๊ก **Pull requests**
   - **เอาติ๊ก Pushes ออก** (GitHub ติ๊กไว้ให้อัตโนมัติ แต่โปรแกรมไม่ได้ใช้)
4. กด **Add webhook**

**ตรวจสอบ:** กดเข้าไปที่ webhook ที่สร้าง → แท็บ **Recent Deliveries** จะเห็น event `ping`

- เครื่องหมายถูกสีเขียว และได้ `200`: เชื่อมต่อสำเร็จ
- ได้ `401`: Secret ไม่ตรงกับ `.env`
- ได้ `400`: Content type ไม่ใช่ `application/json`
- เชื่อมต่อไม่ได้หรือ timeout: โปรแกรมไม่ได้รัน หรือ GitHub เข้าถึง URL นั้นไม่ได้

แก้แล้วกด **Redeliver** เพื่อส่งใหม่ได้

---

## ส่วนที่ 4: ให้คนในทีมผูกบัญชี

แต่ละคนพิมพ์ในช่องแชตใดก็ได้ของ Discord server ที่มี bot:

```text
/git-notify github:octocat
```

- `octocat` คือ **ชื่อผู้ใช้ GitHub** ของตัวเอง ดูได้จาก URL โปรไฟล์ `https://github.com/octocat` (วางลิงก์ทั้งลิงก์ก็ได้)
- bot จะตอบกลับเป็นข้อความที่เห็นแค่คนพิมพ์ ว่าผูกบัญชีแล้ว
- ถ้าพิมพ์ผิดหรือเปลี่ยนชื่อ GitHub ให้พิมพ์คำสั่งใหม่ ค่าเดิมจะถูกแทนที่

คนที่จะรับ DM ได้ต้อง:

- อยู่ใน Discord server เดียวกับ bot
- เปิดรับ DM จากสมาชิกใน server: คลิกชื่อ server → **Privacy Settings** → เปิด **Direct Messages** (ปกติเปิดอยู่แล้ว)

**ทดสอบ:** ให้คนอื่น assign issue ให้คุณบน GitHub จะได้ DM จาก bot (ถ้า assign ให้ตัวเอง จะไม่ได้ DM เพราะระบบไม่แจ้งเตือนสิ่งที่ตัวเองทำ)

---

## เลือกว่าจะแจ้งเตือนอะไร

ตั้งค่าได้สองชั้น:

1. **GitHub Webhook** (ส่วนที่ 3): เลือกประเภท event คือ **Issues**, **Pull requests** หรือทั้งสองอย่าง
2. **`GITHUB_NOTIFY_ACTIONS`** ใน `.env`: เลือกว่า action ไหนจะส่ง DM คั่นด้วยจุลภาค ค่าเริ่มต้นคือ `assigned,closed,reopened,labeled,unlabeled`

| action | เกิดเมื่อ | ใครได้รับ DM |
| --- | --- | --- |
| `assigned` | มีคนถูก assign | คนที่ถูก assign |
| `unassigned` | มีคนถูกเอาออกจากงาน | คนที่ถูกเอาออก |
| `closed` | ปิด Issue หรือ merge / ปิด PR | ทุกคนที่ถูก assign |
| `reopened` | เปิดใหม่อีกครั้ง | ทุกคนที่ถูก assign |
| `labeled` / `unlabeled` | เพิ่ม / ลบ label เช่นใช้ label เป็นสถานะ `in-progress`, `review` | ทุกคนที่ถูก assign |
| `edited` | แก้ชื่อหรือรายละเอียด | ทุกคนที่ถูก assign |
| `review_requested` | ขอ review PR | ทุกคนที่ถูก assign PR |

คนที่เป็นคนทำการเปลี่ยนแปลงเองจะไม่ได้รับ DM เสมอ แก้ `.env` แล้วต้อง restart โปรแกรม

รายการ action ทั้งหมด: [issues](https://docs.github.com/en/webhooks/webhook-events-and-payloads#issues), [pull_request](https://docs.github.com/en/webhooks/webhook-events-and-payloads#pull_request)

ยังไม่รองรับ: คอลัมน์ Status ใน **GitHub Projects** (Todo → In Progress → Done) และคอมเมนต์ใน Issue

---

## แก้ปัญหาเบื้องต้น

| อาการ | สาเหตุและวิธีแก้ |
| --- | --- |
| รันแล้วขึ้น `DISCORD_TOKEN ไม่ได้ถูกตั้งค่า` | ยังไม่ได้ใส่ token ใน `.env` หรือไม่ได้รันในโฟลเดอร์ที่มี `.env` |
| รันแล้วขึ้น `ไม่สามารถเชื่อมต่อ Discord` หรือ `401 Unauthorized` | token ผิด อาจ copy Client Secret หรือ Public Key มาแทน ให้ Reset Token ในหน้า **Bot** แล้ว copy ใหม่ |
| ขึ้น `ลงทะเบียนคำสั่ง /git-notify ไม่สำเร็จ` | `DISCORD_GUILD_ID` ผิด หรือยังไม่ได้เชิญ bot เข้า server นั้น (ข้อ 1.3) |
| ไม่เห็นคำสั่ง `/git-notify` ใน Discord | ตอนเชิญ bot ไม่ได้ติ๊ก `applications.commands` ให้เชิญใหม่ หรือลองปิดเปิด Discord |
| GitHub ได้ `401` | Secret ใน GitHub ไม่ตรงกับ `GITHUB_WEBHOOK_SECRET` |
| GitHub ได้ `400 Invalid JSON` | Content type ใน GitHub ไม่ใช่ `application/json` |
| GitHub ได้ `200` แต่ไม่มี DM | ดู log ของโปรแกรม (ดูด้านล่าง) |

ข้อความใน log เมื่อ GitHub ได้ `200` แต่ไม่มี DM:

- `ยังไม่ได้ผูกบัญชีด้วย /git-notify`: คนนั้นยังไม่ได้พิมพ์คำสั่ง หรือพิมพ์ชื่อ GitHub ผิด
- `ข้าม event=... action=...`: action นั้นไม่ได้อยู่ใน `GITHUB_NOTIFY_ACTIONS`
- `ส่ง DM ... ไม่สำเร็จ`: คนนั้นปิดรับ DM หรือไม่ได้อยู่ใน server เดียวกับ bot
- ไม่มี log อะไรเลย: Issue นั้นไม่มีคนถูก assign หรือคนที่ถูก assign เป็นคนทำเอง

---

## Deploy อัตโนมัติด้วย GitHub Actions (ไม่บังคับ)

`.github/workflows/deploy.yml` จะรัน test ทุกครั้งที่ push เข้า `main` หรือเปิด PR และจะ deploy เมื่อ push tag รูปแบบ `v1.2.3`

1. **Self-hosted runner** บน server: repo → **Settings** → **Actions** → **Runners** → **New self-hosted runner** ทำตามคำสั่งที่แสดง และเพิ่ม label `prod` เครื่องต้องมี Docker และ Docker Compose และ user ที่รัน runner ต้องใช้คำสั่ง `docker` ได้
2. **Secrets** (repo → **Settings** → **Secrets and variables** → **Actions** → แท็บ **Secrets**):

   | ชื่อ | ค่า |
   | --- | --- |
   | `DISCORD_TOKEN` | เหมือน `DISCORD_TOKEN` ใน `.env` |
   | `GH_WEBHOOK_SECRET` | เหมือน `GITHUB_WEBHOOK_SECRET` (GitHub ไม่ให้ตั้งชื่อ secret ขึ้นต้นด้วย `GITHUB_`) |

3. **Variables** (แท็บ **Variables**, ไม่บังคับ):

   | ชื่อ | ค่า |
   | --- | --- |
   | `DISCORD_GUILD_ID` | Server ID |
   | `GITHUB_NOTIFY_ACTIONS` | ถ้าต้องการเปลี่ยนจากค่าเริ่มต้น |
   | `DATA_DIR` | โฟลเดอร์บน server ที่เก็บการผูกบัญชี ค่าเริ่มต้น `/var/lib/git-notify` ต้องอยู่นอกโฟลเดอร์ของ runner เพราะ runner ล้างโฟลเดอร์นั้นทุกครั้งที่ deploy |

4. deploy:

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

---

## สำหรับนักพัฒนา

```bash
go test ./...
```

| ไฟล์ | หน้าที่ |
| --- | --- |
| `main.go` | โหลด `.env`, เชื่อมต่อ Discord, ลงทะเบียนคำสั่ง และเริ่ม HTTP server |
| `core/discord.go` | สร้าง Discord session, ลงทะเบียน `/git-notify` และส่ง DM |
| `core/http.go` | ลงทะเบียน route `/webhook/github` |
| `app/discord_command.go` | จัดการคำสั่ง `/git-notify` |
| `app/user_store.go` | เก็บการผูกบัญชี GitHub กับ Discord ลงไฟล์ JSON |
| `app/github_webhook.go` | ตรวจ signature, เลือกคนที่จะได้ DM และสร้างข้อความ |
| `types/github/payload.go` | โครงสร้าง payload ของ GitHub |
